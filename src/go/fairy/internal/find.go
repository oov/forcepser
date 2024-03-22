package internal

import (
	"syscall"
	"unsafe"

	"github.com/zzl/go-win32api/win32"
)

type callbackData struct {
	windowName string
	className  string
	buf        [1024]uint16
	pid        uint32
	h          win32.HWND
	fn         func(h win32.HWND) bool
}

func enumWindowCallback(h win32.HWND, lParam unsafe.Pointer) uintptr {
	data := (*callbackData)(lParam)
	if h == 0 {
		return uintptr(win32.FALSE)
	}
	if win32.IsWindowVisible(h) == win32.FALSE {
		return uintptr(win32.TRUE)
	}
	if data.windowName != "" {
		win32.GetWindowTextW(h, &data.buf[0], int32(len(data.buf)))
		if syscall.UTF16ToString(data.buf[:]) != data.windowName {
			return uintptr(win32.TRUE)
		}
	}
	if data.className != "" {
		win32.GetClassNameW(h, &data.buf[0], int32(len(data.buf)))
		if syscall.UTF16ToString(data.buf[:]) != data.className {
			return uintptr(win32.TRUE)
		}
	}
	if data.pid != 0 {
		var pid uint32
		win32.GetWindowThreadProcessId(h, &pid)
		if pid != data.pid {
			return uintptr(win32.TRUE)
		}
	}
	if data.fn != nil && !data.fn(h) {
		return uintptr(win32.TRUE)
	}
	data.h = h
	return uintptr(win32.FALSE)
}

var enumWindowCallbackPtr = syscall.NewCallback(enumWindowCallback)

func FindWindow(parent win32.HWND, className string, windowName string, pid uint32, test func(h win32.HWND) bool) win32.HWND {
	data := callbackData{
		windowName: windowName,
		className:  className,
		pid:        uint32(pid),
		fn:         test,
	}
	win32.EnumWindows(enumWindowCallbackPtr, uintptr(unsafe.Pointer(&data)))
	return data.h
}
