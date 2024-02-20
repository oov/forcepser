package aivoice2

import (
	"fmt"
	"syscall"

	"github.com/oov/forcepser/fairy/internal"

	"github.com/zzl/go-win32api/win32"
)

type mainWindow struct {
	window *internal.Element
	export *internal.Element
	view   win32.HWND
}

func (mw *mainWindow) Release() {
	if mw.export != nil {
		mw.export.Release()
		mw.export = nil
	}
	if mw.window != nil {
		mw.window.Release()
		mw.window = nil
	}
}

func newMainWindow(uia *internal.UIAutomation, hwnd win32.HWND) (*mainWindow, error) {
	viewClassName, err := syscall.UTF16PtrFromString(mainWindowViewClassName)
	if err != nil {
		return nil, fmt.Errorf("failed to create View class name string: %w", err)
	}
	viewHwnd, err := win32.FindWindowExW(hwnd, 0, viewClassName, nil)
	if viewHwnd == 0 {
		return nil, fmt.Errorf("failed to find view window: %w", err)
	}

	var window *internal.Element
	{
		// verify window
		var conds []*win32.IUIAutomationCondition
		cndCtrl, err := uia.CreateInt32PropertyCondition(win32.UIA_ControlTypePropertyId, win32.UIA_WindowControlTypeId)
		if err != nil {
			return nil, fmt.Errorf("failed to create control type condition: %w", err)
		}
		defer cndCtrl.Release()
		conds = append(conds, cndCtrl)

		cndClassName, err := uia.CreateStringPropertyConditionEx(win32.UIA_ClassNamePropertyId, mainWindowClassName, 0)
		if err != nil {
			return nil, fmt.Errorf("failed to create class name condition: %w", err)
		}
		defer cndClassName.Release()
		conds = append(conds, cndClassName)

		cndFramework, err := uia.CreateStringPropertyConditionEx(win32.UIA_FrameworkIdPropertyId, mainWindowFramework, 0)
		if err != nil {
			return nil, fmt.Errorf("failed to create framework condition: %w", err)
		}
		defer cndFramework.Release()
		conds = append(conds, cndFramework)

		cond, err := uia.CreateAndCondition(conds...)
		if err != nil {
			return nil, fmt.Errorf("failed to create and condition: %w", err)
		}
		defer cond.Release()

		if hwnd != 0 && hwnd != win32.INVALID_HANDLE_VALUE {
			elem, err := uia.ElementFromHandle(hwnd)
			if err != nil {
				return nil, fmt.Errorf("failed to create element from native window handle: %w", err)
			}
			defer elem.Release()
			window, err = elem.FindFirst(win32.TreeScope_Element, cond)
			if err != nil {
				return nil, fmt.Errorf("FindFirst failed: %w", err)
			}
		} else {
			window, err = uia.FindTopElement(cond)
			if err != nil {
				return nil, fmt.Errorf("FindFirst failed: %w", err)
			}
		}
		defer window.Release()
	}

	// find export button
	var export *internal.Element
	{
		var conds []*win32.IUIAutomationCondition

		cndCtrl, err := uia.CreateInt32PropertyCondition(win32.UIA_ControlTypePropertyId, win32.UIA_ButtonControlTypeId)
		if err != nil {
			return nil, fmt.Errorf("failed to create control type condition: %w", err)
		}
		defer cndCtrl.Release()
		conds = append(conds, cndCtrl)

		var nameConds []*win32.IUIAutomationCondition
		for _, s := range mainWindowExportButtonCaptions {
			cond, err := uia.CreateStringPropertyConditionEx(win32.UIA_NamePropertyId, s, 0)
			if err != nil {
				return nil, fmt.Errorf("failed to create name condition: %w", err)
			}
			defer cond.Release()
			nameConds = append(nameConds, cond)
		}
		orCond, err := uia.CreateOrCondition(nameConds...)
		if err != nil {
			return nil, fmt.Errorf("failed to create or condition: %w", err)
		}
		defer orCond.Release()
		conds = append(conds, orCond)

		andCond, err := uia.CreateAndCondition(conds...)
		if err != nil {
			return nil, fmt.Errorf("failed to create and condition: %w", err)
		}
		defer andCond.Release()

		export, err = window.FindFirst(win32.TreeScope_Descendants, andCond)
		if err != nil {
			return nil, fmt.Errorf("failed to find first: %w", err)
		}
		defer export.Release()
	}

	window.AddRef()
	export.AddRef()
	return &mainWindow{
		window: window,
		export: export,
		view:   viewHwnd,
	}, nil
}
