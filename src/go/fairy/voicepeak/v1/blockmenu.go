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
	menuWindow, err := findSubWindow(uia, pid, mainWindow)
	if err != nil {
		return nil, fmt.Errorf("menu window not found: %w", err)
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
	if elems.Len != 2 {
		return nil, fmt.Errorf("unexpected number of menu item: %d", elems.Len)
	}
	menuItem, err := elems.Get(0)
	if err != nil {
		return nil, fmt.Errorf("failed to get menu item: %w", err)
	}
	defer menuItem.Release()
	name, err := menuItem.GetName()
	if err != nil {
		return nil, fmt.Errorf("failed to get menu item name: %w", err)
	}
	if !match(name, blockExportMenuItemCaptions) {
		return nil, fmt.Errorf("unexpected menu item name: %q", name)
	}

	err = menuWindow.SetEnable(false)
	if err != nil {
		return nil, fmt.Errorf("failed to disable window")
	}

	menuWindow.AddRef()
	menuItem.AddRef()
	return &blockMenu{
		window: menuWindow,
		export: menuItem,
	}, nil
}
