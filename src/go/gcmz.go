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

func openFileMapping(desiredAccess uint32, inheritHandle uint32, name *uint16) (handle syscall.Handle, err error) {
	r0, _, e1 := syscall.Syscall(procOpenFileMappingW.Addr(), 3, uintptr(desiredAccess), uintptr(inheritHandle), uintptr(unsafe.Pointer(name)))
	handle = syscall.Handle(r0)
	if handle == 0 {
		if e1 != 0 {
			err = e1
		} else {
			err = syscall.EINVAL
		}
	}
	return
}

func getConsoleWindow() (handle syscall.Handle) {
	r0, _, _ := syscall.Syscall(procGetConsoleWindow.Addr(), 0, 0, 0, 0)
	handle = syscall.Handle(r0)
	return
}

func sendMessage(hwnd syscall.Handle, uMsg uint32, wParam uintptr, lParam uintptr) (lResult uintptr) {
	r0, _, _ := syscall.Syscall6(procSendMessageW.Addr(), 4, uintptr(hwnd), uintptr(uMsg), uintptr(wParam), uintptr(lParam), 0, 0)
	lResult = uintptr(r0)
	return
}

type gcmzDropsData struct {
	Window     syscall.Handle
	Width      int
	Height     int
	VideoRate  int
	VideoScale int
	AudioRate  int
	AudioCh    int
}

func readGCMZDropsData() gcmzDropsData {
	fileMappingName, err := syscall.UTF16PtrFromString("GCMZDrops")
	if err != nil {
		return gcmzDropsData{}
	}

	fmo, err := openFileMapping(syscall.FILE_MAP_READ, 0, fileMappingName)
	if err != nil {
		return gcmzDropsData{}
	}
	defer syscall.CloseHandle(fmo)

	p, err := syscall.MapViewOfFile(fmo, syscall.FILE_MAP_READ, 0, 0, 0)
	if err != nil {
		return gcmzDropsData{}
	}
	defer syscall.UnmapViewOfFile(p)

	var m []byte
	mh := (*reflect.SliceHeader)(unsafe.Pointer(&m))
	mh.Data = p
	mh.Len = 28
	mh.Cap = mh.Len
	return gcmzDropsData{
		Window:     syscall.Handle(binary.LittleEndian.Uint32(m[0:])),
		Width:      int(int32(binary.LittleEndian.Uint32(m[4:]))),
		Height:     int(int32(binary.LittleEndian.Uint32(m[8:]))),
		VideoRate:  int(int32(binary.LittleEndian.Uint32(m[12:]))),
		VideoScale: int(int32(binary.LittleEndian.Uint32(m[16:]))),
		AudioRate:  int(int32(binary.LittleEndian.Uint32(m[20:]))),
		AudioCh:    int(int32(binary.LittleEndian.Uint32(m[24:]))),
	}
}

func luaReadProject(L *lua.LState) int {
	d := readGCMZDropsData()
	if d.Width == 0 {
		return 0
	}

	t := L.NewTable()
	t.RawSetString("window", lua.LNumber(d.Window))
	t.RawSetString("width", lua.LNumber(d.Width))
	t.RawSetString("height", lua.LNumber(d.Height))
	t.RawSetString("video_rate", lua.LNumber(d.VideoRate))
	t.RawSetString("video_scale", lua.LNumber(d.VideoScale))
	t.RawSetString("audio_rate", lua.LNumber(d.AudioRate))
	t.RawSetString("audio_ch", lua.LNumber(d.AudioCh))
	L.Push(t)
	return 1
}

func luaSendFile(L *lua.LState) int {
	window := L.ToInt(1)
	layer := L.ToInt(2)
	frameAdv := L.ToInt(3)
	files := L.ToTable(4)

	dir, err := os.Getwd()
	if err != nil {
		panic("os.Getwd failed: " + err.Error())
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
	sendMessage(syscall.Handle(window), wmCopyData, uintptr(getConsoleWindow()), uintptr(unsafe.Pointer(cds)))
	return 0
}
