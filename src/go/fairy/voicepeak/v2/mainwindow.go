package voicepeak

import (
	"errors"
	"fmt"
	"log"

	"github.com/oov/forcepser/fairy/internal"

	"github.com/zzl/go-win32api/win32"
)

type mainWindow struct {
	window *internal.Element
	combo  *internal.Element
	edit   *internal.Element
	button *internal.Element
}

func (mw *mainWindow) Release() {
	if mw.window != nil {
		mw.window.Release()
		mw.window = nil
	}
	if mw.combo != nil {
		mw.combo.Release()
		mw.combo = nil
	}
	if mw.edit != nil {
		mw.edit.Release()
		mw.edit = nil
	}
	if mw.button != nil {
		mw.button.Release()
		mw.button = nil
	}
}

func findMainWindowControls(elems *internal.Elements, index int, out *mainWindow) error {
	if index < 2 || index >= elems.Len {
		return internal.ErrElementNotFound
	}

	editElem, err := elems.Get(index)
	if err != nil {
		return fmt.Errorf("failed to get edit element: %w", err)
	}
	defer editElem.Release()

	ctrlType, err := editElem.GetControlType()
	if err != nil {
		return fmt.Errorf("failed to get edit control type: %w", err)
	}
	if ctrlType != win32.UIA_EditControlTypeId && ctrlType != win32.UIA_TextControlTypeId {
		return internal.ErrElementNotFound
	}

	buttonElem, err := elems.Get(index - 1)
	if err != nil {
		return fmt.Errorf("failed to get image button element: %w", err)
	}
	defer buttonElem.Release()

	ctrlType, err = buttonElem.GetControlType()
	if err != nil {
		return fmt.Errorf("failed to get image button control type: %w", err)
	}
	if ctrlType != win32.UIA_ButtonControlTypeId {
		return internal.ErrElementNotFound
	}
	name, err := buttonElem.GetName()
	if err != nil {
		return fmt.Errorf("failed to get image button control name: %w", err)
	}
	if name != mainWindowIconButtonName {
		return internal.ErrElementNotFound
	}

	comboElem, err := elems.Get(index - 2)
	if err != nil {
		return fmt.Errorf("failed to get combo box element: %w", err)
	}
	defer comboElem.Release()

	ctrlType, err = comboElem.GetControlType()
	if err != nil {
		return fmt.Errorf("failed to get combo box control type: %w", err)
	}
	if ctrlType != win32.UIA_ButtonControlTypeId {
		return internal.ErrElementNotFound
	}

	comboElem.AddRef()
	out.combo = comboElem
	editElem.AddRef()
	out.edit = editElem
	buttonElem.AddRef()
	out.button = buttonElem
	return nil
}

func findMainWindowControls2(elems *internal.Elements, index int, out *mainWindow) error {
	if index < 3 || index >= elems.Len {
		return internal.ErrElementNotFound
	}

	editElem, err := elems.Get(index)
	if err != nil {
		return fmt.Errorf("failed to get edit element: %w", err)
	}
	defer editElem.Release()

	ctrlType, err := editElem.GetControlType()
	if err != nil {
		return fmt.Errorf("failed to get edit control type: %w", err)
	}
	if ctrlType != win32.UIA_EditControlTypeId && ctrlType != win32.UIA_TextControlTypeId {
		return internal.ErrElementNotFound
	}

	buttonElem, err := elems.Get(index - 2)
	if err != nil {
		return fmt.Errorf("failed to get image button element: %w", err)
	}
	defer buttonElem.Release()

	ctrlType, err = buttonElem.GetControlType()
	if err != nil {
		return fmt.Errorf("failed to get image button control type: %w", err)
	}
	if ctrlType != win32.UIA_ButtonControlTypeId {
		return internal.ErrElementNotFound
	}
	name, err := buttonElem.GetName()
	if err != nil {
		return fmt.Errorf("failed to get image button control name: %w", err)
	}
	if name != mainWindowIconButtonName {
		return internal.ErrElementNotFound
	}

	comboElem, err := elems.Get(index - 3)
	if err != nil {
		return fmt.Errorf("failed to get combo box element: %w", err)
	}
	defer comboElem.Release()

	ctrlType, err = comboElem.GetControlType()
	if err != nil {
		return fmt.Errorf("failed to get combo box control type: %w", err)
	}
	if ctrlType != win32.UIA_ButtonControlTypeId {
		return internal.ErrElementNotFound
	}

	comboElem.AddRef()
	out.combo = comboElem
	editElem.AddRef()
	out.edit = editElem
	buttonElem.AddRef()
	out.button = buttonElem
	return nil
}

func newMainWindow(uia *internal.UIAutomation, hwnd win32.HWND) (*mainWindow, error) {
	// verify window
	var conds []*win32.IUIAutomationCondition
	cndCtrl, err := uia.CreateInt32PropertyCondition(win32.UIA_ControlTypePropertyId, win32.UIA_WindowControlTypeId)
	if err != nil {
		return nil, fmt.Errorf("failed to create control type condition: %w", err)
	}
	defer cndCtrl.Release()
	conds = append(conds, cndCtrl)

	cndName, err := uia.CreateStringPropertyConditionEx(win32.UIA_NamePropertyId, mainWindowName, 0)
	if err != nil {
		return nil, fmt.Errorf("failed to create name condition: %w", err)
	}
	defer cndName.Release()
	conds = append(conds, cndName)

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

	var window *internal.Element
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

	// find controls
	cndTrue, err := uia.CreateTrueCondition()
	if err != nil {
		return nil, fmt.Errorf("failed to create true condition: %w", err)
	}
	defer cndTrue.Release()

	elems, err := window.FindAll(win32.TreeScope_Children, cndTrue)
	if err != nil {
		return nil, fmt.Errorf("failed to get children: %w", err)
	}
	defer elems.Release()

	r := mainWindow{
		window: window,
	}
	for i := 0; i < elems.Len && r.edit == nil; i++ {
		err = findMainWindowControls(elems, i, &r)
		if err != nil {
			if !errors.Is(err, internal.ErrElementNotFound) {
				log.Printf("failed to get elements: %v", err)
			}
		}
		if err == nil {
			continue
		}
		err = findMainWindowControls2(elems, i, &r)
		if err != nil {
			if !errors.Is(err, internal.ErrElementNotFound) {
				log.Printf("failed to get elements: %v", err)
			}
		}
	}
	if r.combo == nil || r.edit == nil || r.button == nil {
		return nil, fmt.Errorf("main window controls not found")
	}
	defer r.combo.Release()
	defer r.edit.Release()
	defer r.button.Release()

	window.AddRef()
	r.combo.AddRef()
	r.edit.AddRef()
	r.button.AddRef()
	return &r, nil
}
