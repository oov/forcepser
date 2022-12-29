package internal

import (
	"fmt"
	"syscall"
	"unsafe"

	"github.com/zzl/go-win32api/win32"
)

type UIAutomation struct {
	*win32.IUIAutomation
}

func New() (*UIAutomation, error) {
	var CLSID_CUIAutomation = syscall.GUID{
		Data1: 0xFF48DBA4,
		Data2: 0x60EF,
		Data3: 0x4201,
		Data4: [8]byte{0xAA, 0x87, 0x54, 0x10, 0x3E, 0xEF, 0x59, 0x4E},
	}
	var uia *win32.IUIAutomation
	if hr := win32.CoCreateInstance(&CLSID_CUIAutomation, nil, win32.CLSCTX_INPROC_SERVER, &win32.IID_IUIAutomation, unsafe.Pointer(&uia)); win32.FAILED(hr) || uia == nil {
		return nil, fmt.Errorf("failed to create IUIAutomation: %s", win32.HRESULT_ToString(hr))
	}
	return &UIAutomation{uia}, nil
}

func (uia *UIAutomation) ElementFromHandle(hwnd win32.HWND) (*Element, error) {
	var elem *win32.IUIAutomationElement
	if hr := uia.IUIAutomation.ElementFromHandle(hwnd, &elem); win32.FAILED(hr) || elem == nil {
		return nil, fmt.Errorf("IUIAutomation.ElementFromHandle failed: %s", win32.HRESULT_ToString(hr))
	}
	return &Element{elem}, nil
}

func (uia *UIAutomation) GetRootElement() (*Element, error) {
	var elem *win32.IUIAutomationElement
	if hr := uia.IUIAutomation.GetRootElement(&elem); win32.FAILED(hr) || elem == nil {
		return nil, fmt.Errorf("IUIAutomation.GetRootElement failed: %s", win32.HRESULT_ToString(hr))
	}
	return &Element{elem}, nil
}

func (uia *UIAutomation) FindTopElement(conds ...*win32.IUIAutomationCondition) (*Element, error) {
	cond, err := uia.CreateAndCondition(conds...)
	if err != nil {
		return nil, fmt.Errorf("failed to create and condition: %w", err)
	}
	defer cond.Release()

	root, err := uia.GetRootElement()
	if err != nil {
		return nil, fmt.Errorf("failed to get root element: %w", err)
	}
	defer root.Release()

	elem, err := root.FindFirst(win32.TreeScope_Children, cond)
	if err != nil {
		return nil, fmt.Errorf("failed to find first: %w", err)
	}
	return elem, nil
}
