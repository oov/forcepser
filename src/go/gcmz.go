package main

import (
	"encoding/binary"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"syscall"
	"unicode/utf16"
	"unsafe"

	lua "github.com/yuin/gopher-lua"
	"golang.org/x/sys/windows"
)

var modKernel32 = windows.NewLazySystemDLL("kernel32.dll")
var modUser32 = windows.NewLazySystemDLL("user32.dll")

var procOpenFileMappingW = modKernel32.NewProc("OpenFileMappingW")
var procGetConsoleWindow = modKernel32.NewProc("GetConsoleWindow")
var procSendMessageW = modUser32.NewProc("SendMessageW")

func openFileMapping(desiredAccess uint32, inheritHandle uint32, name *uint16) (handle windows.Handle, err error) {
	r0, _, e1 := syscall.Syscall(procOpenFileMappingW.Addr(), 3, uintptr(desiredAccess), uintptr(inheritHandle), uintptr(unsafe.Pointer(name)))
	handle = windows.Handle(r0)
	if handle == 0 {
		if e1 != 0 {
			err = e1
		} else {
			err = syscall.EINVAL
		}
	}
	return
}

func getConsoleWindow() (handle windows.Handle) {
	r0, _, _ := syscall.Syscall(procGetConsoleWindow.Addr(), 0, 0, 0, 0)
	handle = windows.Handle(r0)
	return
}

func sendMessage(hwnd windows.Handle, uMsg uint32, wParam uintptr, lParam uintptr) (lResult uintptr) {
	r0, _, _ := syscall.Syscall6(procSendMessageW.Addr(), 4, uintptr(hwnd), uintptr(uMsg), uintptr(wParam), uintptr(lParam), 0, 0)
	lResult = uintptr(r0)
	return
}

type gcmzDropsData struct {
	Window      windows.Handle
	Width       int
	Height      int
	VideoRate   int
	VideoScale  int
	AudioRate   int
	AudioCh     int
	GCMZAPIVer  int
	ProjectFile string
}

func readGCMZDropsData() (*gcmzDropsData, error) {
	fileMappingName, err := windows.UTF16PtrFromString("GCMZDrops")
	if err != nil {
		return nil, err
	}
	mutexName, err := windows.UTF16PtrFromString("GCMZDropsMutex")
	if err != nil {
		return nil, err
	}

	fmo, err := openFileMapping(windows.FILE_MAP_READ, 0, fileMappingName)
	if err != nil {
		return nil, err
	}
	defer windows.CloseHandle(fmo)

	p, err := windows.MapViewOfFile(fmo, windows.FILE_MAP_READ, 0, 0, 0)
	if err != nil {
		return nil, err
	}
	defer windows.UnmapViewOfFile(p)

	var oldAPI = false
	mutex, err := windows.OpenMutex(windows.MUTEX_ALL_ACCESS, false, mutexName)
	if err != nil {
		oldAPI = true
	} else {
		defer windows.CloseHandle(mutex)
		windows.WaitForSingleObject(mutex, windows.INFINITE)
		defer windows.ReleaseMutex(mutex)
	}

	var m []byte
	mh := (*reflect.SliceHeader)(unsafe.Pointer(&m))
	mh.Data = p
	mh.Len = 32 + windows.MAX_PATH
	mh.Cap = mh.Len
	r := &gcmzDropsData{
		Window:     windows.Handle(binary.LittleEndian.Uint32(m[0:])),
		Width:      int(int32(binary.LittleEndian.Uint32(m[4:]))),
		Height:     int(int32(binary.LittleEndian.Uint32(m[8:]))),
		VideoRate:  int(int32(binary.LittleEndian.Uint32(m[12:]))),
		VideoScale: int(int32(binary.LittleEndian.Uint32(m[16:]))),
		AudioRate:  int(int32(binary.LittleEndian.Uint32(m[20:]))),
		AudioCh:    int(int32(binary.LittleEndian.Uint32(m[24:]))),
	}
	if !oldAPI {
		r.GCMZAPIVer = int(int32(binary.LittleEndian.Uint32(m[28:])))
		r.ProjectFile = windows.UTF16PtrToString((*uint16)(unsafe.Pointer(&m[32])))
	}
	return r, nil
}

func luaSendFile(L *lua.LState) int {
	window := L.ToInt(1)
	layer := L.ToInt(2)
	frameAdv := L.ToInt(3)
	files := L.ToTable(4)

	dir, err := os.Getwd()
	if err != nil {
		L.RaiseError("os.Getwd failed: %v", err)
	}

	buf := make([]byte, 0, 64)
	buf = append(buf, strconv.Itoa(layer)...)
	buf = append(buf, 0x00)
	buf = append(buf, strconv.Itoa(frameAdv)...)

	n := files.MaxN()
	for i := 1; i <= n; i++ {
		buf = append(buf, 0x00)
		buf = append(buf, filepath.Join(dir, files.RawGetInt(i).String())...)
	}

	str := utf16.Encode([]rune(string(buf)))

	const wmCopyData = 0x4A
	type copyDataStruct struct {
		Data uintptr
		Size uint32
		Ptr  uintptr
	}
	cds := &copyDataStruct{
		Data: 0,
		Size: uint32(len(str) * 2),
		Ptr:  uintptr(unsafe.Pointer(&str[0])),
	}
	sendMessage(windows.Handle(window), wmCopyData, uintptr(getConsoleWindow()), uintptr(unsafe.Pointer(cds)))
	return 0
}
