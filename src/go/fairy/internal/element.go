package internal

import (
	"errors"
	"fmt"
	"syscall"
	"unsafe"

	"github.com/zzl/go-win32api/win32"
	"golang.org/x/sys/windows"
)

var (
	ErrElementNotFound = fmt.Errorf("element not found")
	ErrAborted         = fmt.Errorf("aborted")
)

type Element struct {
	*win32.IUIAutomationElement
}

func (elem *Element) GetCurrentPropertyInt32Value(propertyID int32) (int32, error) {
	var v win32.VARIANT
	if hr := elem.GetCurrentPropertyValue(propertyID, &v); win32.FAILED(hr) {
		return 0, fmt.Errorf("IUIAutomationElement.GetCurrentPropertyValue failed: %s", win32.HRESULT_ToString(hr))
	}
	defer win32.VariantClear(&v)
	value, err := variantToInt32(&v)
	if err != nil {
		return 0, fmt.Errorf("failed convert value: %w", err)
	}
	return value, nil
}

func (elem *Element) GetCurrentPropertyStringValue(propertyID int32) (string, error) {
	var v win32.VARIANT
	if hr := elem.GetCurrentPropertyValue(propertyID, &v); win32.FAILED(hr) {
		return "", fmt.Errorf("IUIAutomationElement.GetCurrentPropertyStringValue failed: %s", win32.HRESULT_ToString(hr))
	}
	defer win32.VariantClear(&v)
	value := win32.BstrToStr(v.BstrValVal())
	return value, nil
}

func (elem *Element) GetCurrentPropertyRectValue(propertyID int32) ([4]float64, error) {
	var v win32.VARIANT
	if hr := elem.GetCurrentPropertyValue(propertyID, &v); win32.FAILED(hr) {
		return [4]float64{}, fmt.Errorf("IUIAutomationElement.GetCurrentPropertyStringValue failed: %s", win32.HRESULT_ToString(hr))
	}
	defer win32.VariantClear(&v)
	var value [4]float64
	for i := int32(0); i < 4; i++ {
		var f float64
		if hr := win32.SafeArrayGetElement(v.ParrayVal(), &i, unsafe.Pointer(&f)); win32.FAILED(hr) {
			return [4]float64{}, fmt.Errorf("SafeArrayGetElement failed: %s", win32.HRESULT_ToString(hr))
		}
		value[i] = f
	}
	return value, nil
}
func (elem *Element) FindFirst(scope win32.TreeScope, cond *win32.IUIAutomationCondition) (*Element, error) {
	var found *win32.IUIAutomationElement
	if hr := elem.IUIAutomationElement.FindFirst(scope, cond, &found); win32.FAILED(hr) {
		return nil, fmt.Errorf("IUIAutomationElement.FindFirst failed: %s", win32.HRESULT_ToString(hr))
	}
	if found == nil {
		return nil, ErrElementNotFound
	}
	return &Element{found}, nil
}

func (elem *Element) FindAll(scope win32.TreeScope, cond *win32.IUIAutomationCondition) (*Elements, error) {
	var elems *win32.IUIAutomationElementArray
	if hr := elem.IUIAutomationElement.FindAll(scope, cond, &elems); win32.FAILED(hr) {
		return nil, fmt.Errorf("IUIAutomationElement.FindAll failed: %s", win32.HRESULT_ToString(hr))
	}
	if elems == nil {
		return nil, ErrElementNotFound
	}
	defer elems.Release()

	var n int32
	if hr := elems.Get_Length(&n); win32.FAILED(hr) {
		return nil, fmt.Errorf("IUIAutomationElement.Get_Length failed: %s", win32.HRESULT_ToString(hr))
	}
	elems.AddRef()
	return &Elements{elems, int(n)}, nil
}

func (elem *Element) GetName() (string, error) {
	var v win32.BSTR
	if hr := elem.Get_CurrentName(&v); win32.FAILED(hr) {
		return "", fmt.Errorf("IUIAutomationElement.Get_CurrentName failed: %s", win32.HRESULT_ToString(hr))
	}
	return win32.BstrToStrAndFree(v), nil
}

func (elem *Element) GetControlType() (int32, error) {
	var v int32
	if hr := elem.Get_CurrentControlType(&v); win32.FAILED(hr) {
		return 0, fmt.Errorf("IUIAutomationElement.Get_CurrentControlType failed: %s", win32.HRESULT_ToString(hr))
	}
	return v, nil
}

func (elem *Element) GetProcessID() (win32.DWORD, error) {
	var v int32
	if hr := elem.Get_CurrentProcessId(&v); win32.FAILED(hr) {
		return 0, fmt.Errorf("IUIAutomationElement.Get_CurrentProcessId failed: %s", win32.HRESULT_ToString(hr))
	}
	return win32.DWORD(v), nil
}

func (elem *Element) GetNativeWindowHandle() (win32.HWND, error) {
	var v win32.HWND
	if hr := elem.Get_CurrentNativeWindowHandle(&v); win32.FAILED(hr) {
		return win32.INVALID_HANDLE_VALUE, fmt.Errorf("IUIAutomationElement.Get_CurrentNativeWindowHandle failed: %s", win32.HRESULT_ToString(hr))
	}
	return v, nil
}

func (elem *Element) SetEnable(enable bool) error {
	hwnd, err := elem.GetNativeWindowHandle()
	if err != nil {
		return fmt.Errorf("failed to get native window handle: %w", err)
	}
	var b win32.BOOL
	if enable {
		b = win32.TRUE
	} else {
		b = win32.FALSE
	}
	win32.EnableWindow(hwnd, b)
	return nil
}

func (elem *Element) GetModulePath() (string, error) {
	pid, err := elem.GetProcessID()
	if err != nil {
		return "", fmt.Errorf("failed to get process id: %w", err)
	}
	s, err := GetModulePathFromPID(pid)
	if err != nil {
		return "", fmt.Errorf("failed to get module path: %w", err)
	}
	return s, nil
}

func GetModulePathFromPID(pid uint32) (string, error) {
	h, werr := win32.OpenProcess(win32.PROCESS_QUERY_INFORMATION|win32.PROCESS_VM_READ, win32.FALSE, pid)
	if h == 0 {
		return "", fmt.Errorf("failed to open process: %w", werr)
	}
	defer win32.CloseHandle(h)

	var l win32.DWORD
	var buf []uint16
	var err error
	for {
		l += win32.MAX_PATH
		buf = make([]uint16, l)
		err = windows.GetModuleFileNameEx(windows.Handle(h), 0, &buf[0], l)
		if errors.Is(err, windows.ERROR_INSUFFICIENT_BUFFER) {
			continue
		}
		break
	}
	if err != nil {
		return "", err
	}
	return syscall.UTF16ToString(buf), nil
}

func GetWindowText(hwnd win32.HWND) string {
	n, _ := win32.SendMessageW(hwnd, win32.WM_GETTEXTLENGTH, 0, 0)
	if n == 0 {
		return ""
	}
	buf := make([]uint16, n+1)
	n, _ = win32.SendMessageW(hwnd, win32.WM_GETTEXT, uintptr(n+1), uintptr(unsafe.Pointer(&buf[0])))
	if n == 0 {
		return ""
	}
	return syscall.UTF16ToString(buf[:n])
}
