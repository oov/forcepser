package voicepeak

import (
	"fmt"
	"path/filepath"

	"github.com/oov/forcepser/fairy/internal"

	"github.com/zzl/go-win32api/win32"
)

func match(s string, patterns []string) bool {
	for _, ps := range patterns {
		if s == ps {
			return true
		}
	}
	return false
}

func changeExt(path, ext string) string {
	return path[:len(path)-len(filepath.Ext(path))] + ext
}

func findSubWindow(uia *internal.UIAutomation, pid win32.DWORD, mainWindow win32.HWND, additionalConds ...*win32.IUIAutomationCondition) (*internal.Element, error) {
	var conds []*win32.IUIAutomationCondition
	cndCtrl, err := uia.CreateInt32PropertyCondition(win32.UIA_ControlTypePropertyId, win32.UIA_WindowControlTypeId)
	if err != nil {
		return nil, fmt.Errorf("failed to create control type condition: %w", err)
	}
	defer cndCtrl.Release()
	conds = append(conds, cndCtrl)

	if pid != 0 {
		cndProcessID, err := uia.CreateInt32PropertyCondition(win32.UIA_ProcessIdPropertyId, int32(pid))
		if err != nil {
			return nil, fmt.Errorf("failed to create process id condition: %w", err)
		}
		defer cndProcessID.Release()
		conds = append(conds, cndProcessID)
	}

	if mainWindow != 0 && mainWindow != win32.INVALID_HANDLE_VALUE {
		cndWindowHandle, err := uia.CreateInt32PropertyCondition(win32.UIA_NativeWindowHandlePropertyId, int32(mainWindow))
		if err != nil {
			return nil, fmt.Errorf("failed to create native window handle condition: %w", err)
		}
		defer cndWindowHandle.Release()
		cndNotWindowHandle, err := uia.CreateNotCondition(cndWindowHandle)
		if err != nil {
			return nil, fmt.Errorf("failed to create not condition: %w", err)
		}
		defer cndNotWindowHandle.Release()
		conds = append(conds, cndNotWindowHandle)
	}

	conds = append(conds, additionalConds...)

	window, err := uia.FindTopElement(conds...)
	if err != nil {
		return nil, fmt.Errorf("FindFirst failed: %w", err)
	}
	return window, nil
}
