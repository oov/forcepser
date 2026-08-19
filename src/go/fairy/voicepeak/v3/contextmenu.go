package voicepeak

import (
	"fmt"

	"github.com/oov/forcepser/fairy/internal"

	"github.com/zzl/go-win32api/win32"
)

type blockMenu struct {
	window *internal.Element
	export *internal.Element
}

func (bm *blockMenu) Release() {
	if bm.window != nil {
		bm.window.SetEnable(true)
		bm.window.Release()
		bm.window = nil
	}
	if bm.export != nil {
		bm.export.Release()
		bm.export = nil
	}
}

func findBlockMenu(uia *internal.UIAutomation, pid win32.DWORD, mainWindow win32.HWND) (*blockMenu, error) {
	menuWindowHandle := internal.FindWindow(0, "", "", uint32(pid), func(h win32.HWND) bool {
		return h != mainWindow
	})
	if menuWindowHandle == 0 {
		return nil, fmt.Errorf("menu window not found")
	}

	menuWindow, err := uia.ElementFromHandle(menuWindowHandle)
	if err != nil {
		return nil, fmt.Errorf("failed to get menu window: %w", err)
	}
	defer menuWindow.Release()

	cndMenuItem, err := uia.CreateInt32PropertyCondition(win32.UIA_ControlTypePropertyId, win32.UIA_MenuItemControlTypeId)
	if err != nil {
		return nil, fmt.Errorf("failed to create cond: %w", err)
	}
	defer cndMenuItem.Release()

	elems, err := menuWindow.FindAll(win32.TreeScope_Children, cndMenuItem)
	if err != nil {
		return nil, fmt.Errorf("failed to get menu items: %w", err)
	}
	defer elems.Release()
	var found *internal.Element
	for i := elems.Len - 1; i >= 0; i-- {
		menuItem, err := elems.Get(i)
		if err != nil {
			return nil, fmt.Errorf("failed to get menu item: %w", err)
		}
		defer menuItem.Release()
		name, err := menuItem.GetName()
		if err != nil {
			return nil, fmt.Errorf("failed to get menu item name: %w", err)
		}
		if match(name, blockExportMenuItemCaptions) {
			found = menuItem
			break
		}
	}
	if found == nil {
		return nil, fmt.Errorf("menu item not found")
	}

	err = menuWindow.SetEnable(false)
	if err != nil {
		return nil, fmt.Errorf("failed to disable window")
	}

	menuWindow.AddRef()
	found.AddRef()
	return &blockMenu{
		window: menuWindow,
		export: found,
	}, nil
}
