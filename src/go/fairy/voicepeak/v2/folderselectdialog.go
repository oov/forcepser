package voicepeak

import (
	"fmt"

	"github.com/oov/forcepser/fairy/internal"

	"github.com/zzl/go-win32api/win32"
)

type folderSelectDialog struct {
	window *internal.Element
	edit   *internal.Element
	button *internal.Element
}

func (fsd *folderSelectDialog) Release() {
	if fsd.window != nil {
		fsd.window.Release()
		fsd.window = nil
	}
	if fsd.edit != nil {
		fsd.edit.Release()
		fsd.edit = nil
	}
	if fsd.button != nil {
		fsd.button.Release()
		fsd.button = nil
	}
}

func findLegacyControl(uia *internal.UIAutomation, window *internal.Element, controlType int32, controlID int) (*internal.Element, error) {
	ctrlCond, err := uia.CreateInt32PropertyCondition(win32.UIA_ControlTypePropertyId, controlType)
	if err != nil {
		return nil, fmt.Errorf("failed to create control type condition: %w", err)
	}
	defer ctrlCond.Release()

	idCond, err := uia.CreateStringPropertyConditionEx(win32.UIA_AutomationIdPropertyId, fmt.Sprint(controlID), 0)
	if err != nil {
		return nil, fmt.Errorf("failed to create id condition: %w", err)
	}
	defer idCond.Release()

	cond, err := uia.CreateAndCondition(ctrlCond, idCond)
	if err != nil {
		return nil, fmt.Errorf("failed to create and condition: %w", err)
	}
	defer cond.Release()

	elem, err := window.FindFirst(win32.TreeScope_Children, cond)
	if err != nil {
		return nil, fmt.Errorf("element not found: %w", err)
	}
	return elem, nil
}

func findFolderSelectDialog(uia *internal.UIAutomation, pid win32.DWORD, mainWindow win32.HWND) (*folderSelectDialog, error) {
	windowHandle := internal.FindWindow(0, folderSelectDialogClass, "", uint32(pid), func(h win32.HWND) bool {
		return h != mainWindow
	})
	if windowHandle == 0 {
		return nil, fmt.Errorf("folder select dialog not found")
	}

	elem, err := uia.ElementFromHandle(windowHandle)
	if err != nil {
		return nil, fmt.Errorf("failed to get folder select dialog: %w", err)
	}
	defer elem.Release()

	cndFramework, err := uia.CreateStringPropertyConditionEx(win32.UIA_FrameworkIdPropertyId, folderSelectDialogFramework, 0)
	if err != nil {
		return nil, fmt.Errorf("CreateStringPropertyCondition failed: %w", err)
	}
	defer cndFramework.Release()

	dialogElem, err := elem.FindFirst(win32.TreeScope_Element, cndFramework)
	if err != nil {
		return nil, fmt.Errorf("folder select dialog framework not matched: %w", err)
	}
	defer dialogElem.Release()

	editElem, err := findLegacyControl(uia, dialogElem, win32.UIA_EditControlTypeId, folderSelectDialogEditID)
	if err != nil {
		return nil, fmt.Errorf("edit control not found in folder select dialog: %w", err)
	}
	defer editElem.Release()

	buttonElem, err := findLegacyControl(uia, dialogElem, win32.UIA_ButtonControlTypeId, folderSelectDialogButtonID)
	if err != nil {
		return nil, fmt.Errorf("button control not found in folder select dialog: %w", err)
	}
	defer buttonElem.Release()

	dialogElem.AddRef()
	editElem.AddRef()
	buttonElem.AddRef()
	return &folderSelectDialog{
		window: dialogElem,
		edit:   editElem,
		button: buttonElem,
	}, nil
}
