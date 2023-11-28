package internal

import (
	"syscall"
	"unsafe"

	"github.com/zzl/go-win32api/win32"
)

var (
	sendInputProc = syscall.NewLazyDLL("user32.dll").NewProc("SendInput")
)

type KeyboardInput struct {
	Vk        win32.VIRTUAL_KEY
	Scan      uint16
	Flags     uint32
	Time      uint32
	ExtraInfo uint64
}

type Input struct {
	InputType uint32
	KI        KeyboardInput
	Padding   uint64
}

const (
	INPUT_KEYBOARD  = 1
	KEYEVENTF_KEYUP = 0x0002
)

func SendInput(input []Input) (int, error) {
	ret, _, err := sendInputProc.Call(
		uintptr(len(input)),
		uintptr(unsafe.Pointer(&input[0])),
		uintptr(unsafe.Sizeof(input[0])),
	)
	if ret == 0 {
		return 0, err
	}
	return int(ret), nil
}
