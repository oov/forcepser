package main

import (
	"encoding/binary"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"strconv"
	"syscall"
	"unicode/utf16"
	"unsafe"

	lua "github.com/yuin/gopher-lua"
	"golang.org/x/sys/windows"
)

var modKernel32 = windows.NewLazySystemDLL("kernel32.dll")
var modUser32 = windows.NewLazySystemDLL("user32.dll")
var modShell32 = windows.NewLazySystemDLL("shell32.dll")

var procOpenFileMappingW = modKernel32.NewProc("OpenFileMappingW")
var procGetConsoleWindow = modKernel32.NewProc("GetConsoleWindow")
var procSendMessageW = modUser32.NewProc("SendMessageW")
var procSetForegroundWindow = modUser32.NewProc("SetForegroundWindow")
var procSHGetSpecialFolderPath = modShell32.NewProc("SHGetSpecialFolderPathW")

func getFileInfo(path string) (*windows.ByHandleFileInformation, error) {
	name, err := windows.UTF16PtrFromString(path)
	if err != nil {
		return nil, err
	}
	fa, err := windows.GetFileAttributes(name)
	if err != nil {
		return nil, err
	}
	attr := uint32(0)
	if fa&windows.FILE_ATTRIBUTE_DIRECTORY == windows.FILE_ATTRIBUTE_DIRECTORY {
		attr = windows.FILE_FLAG_BACKUP_SEMANTICS
	}
	h, err := windows.CreateFile(name, 0, windows.FILE_SHARE_DELETE|windows.FILE_SHARE_READ|windows.FILE_SHARE_WRITE, nil, windows.OPEN_EXISTING, attr, 0)
	if err != nil {
		return nil, err
	}
	defer windows.CloseHandle(h)
	var fi windows.ByHandleFileInformation
	if err = windows.GetFileInformationByHandle(h, &fi); err != nil {
		return nil, err
	}
	return &fi, nil
}

func isSameFileInfo(fi1 *windows.ByHandleFileInformation, fi2 *windows.ByHandleFileInformation) bool {
	return fi1.VolumeSerialNumber == fi2.VolumeSerialNumber && fi1.FileIndexLow == fi2.FileIndexLow && fi1.FileIndexHigh == fi2.FileIndexHigh
}

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

func sendMessage(hwnd windows.Handle, uMsg uint32, wParam uintptr, lParam uintptr) (lResult uintptr, err error) {
	r0, _, e1 := syscall.Syscall6(procSendMessageW.Addr(), 4, uintptr(hwnd), uintptr(uMsg), uintptr(wParam), uintptr(lParam), 0, 0)
	lResult = uintptr(r0)
	if e1 != 0 {
		err = e1
	}
	return
}

func setForegroundWindow(hwnd windows.Handle) bool {
	r0, _, _ := syscall.Syscall(procSetForegroundWindow.Addr(), 1, uintptr(hwnd), 0, 0)
	return r0 != 0
}

const (
	CSIDL_DESKTOP  = 0x00
	CSIDL_PERSONAL = 0x05
	CSIDL_PROFILE  = 0x28
)

func getSpecialFolderPath(csidl uintptr) string {
	var s [260]uint16
	if !shGetSpecialFolderPath(getConsoleWindow(), &s[0], csidl, false) {
		return ""
	}
	return windows.UTF16ToString(s[:])
}

func shGetSpecialFolderPath(hwnd windows.Handle, path *uint16, csidl uintptr, fCreate bool) bool {
	var b uintptr
	if fCreate {
		b = 1
	}
	r0, _, _ := syscall.Syscall6(procSHGetSpecialFolderPath.Addr(), 4, uintptr(hwnd), uintptr(unsafe.Pointer(path)), csidl, b, 0, 0)
	return r0 != 0
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
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

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
	mh.Len = 32 + windows.MAX_PATH*2
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
	if _, err := sendMessage(windows.Handle(window), wmCopyData, uintptr(getConsoleWindow()), uintptr(unsafe.Pointer(cds))); err != nil {
		L.RaiseError("ごちゃまぜドロップスの外部連携API呼び出しに失敗しました: %v", err)
	}
	return 0
}
