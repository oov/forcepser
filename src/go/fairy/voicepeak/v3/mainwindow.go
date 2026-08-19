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
	if index < 1 || index >= elems.Len {
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

	textPattern, err := editElem.GetCurrentTextPattern()
	if err != nil {
		return internal.ErrElementNotFound
	}
	textPattern.Release()

	keyboardFocusable, err := editElem.GetCurrentPropertyInt32Value(win32.UIA_IsKeyboardFocusablePropertyId)
	if err != nil {
		return fmt.Errorf("failed to get edit keyboard focus property: %w", err)
	}
	if keyboardFocusable == 0 {
		return internal.ErrElementNotFound
	}

	type buttonInfo struct {
		index int
		name  string
	}
	buttons := make([]buttonInfo, 0, index)
	for i := index - 1; i >= 0; i-- {
		elem, err := elems.Get(i)
		if err != nil {
			return fmt.Errorf("failed to get preceding element: %w", err)
		}

		ctrlType, err := elem.GetControlType()
		if err != nil {
			elem.Release()
			return fmt.Errorf("failed to get preceding element control type: %w", err)
		}
		if ctrlType != win32.UIA_ButtonControlTypeId {
			// Never search past a non-button. In particular, this prevents
			// associating this text control with a button belonging to another
			// text control.
			elem.Release()
			break
		}

		name, err := elem.GetName()
		elem.Release()
		if err != nil {
			return fmt.Errorf("failed to get preceding button name: %w", err)
		}
		buttons = append(buttons, buttonInfo{index: i, name: name})
	}
	if len(buttons) == 0 {
		return internal.ErrElementNotFound
	}

	buttonNames := make(map[int]string, len(buttons))
	for _, button := range buttons {
		buttonNames[button.index] = button.name
	}

	comboIndex := -1
	for _, button := range buttons {
		if button.name == "" || button.name == mainWindowIconButtonName {
			continue
		}
		if buttonNames[button.index+1] != mainWindowIconButtonName {
			continue
		}
		if comboIndex != -1 {
			return fmt.Errorf("multiple character button candidates found")
		}
		comboIndex = button.index
	}
	if comboIndex == -1 {
		return internal.ErrElementNotFound
	}

	comboElem, err := elems.Get(comboIndex)
	if err != nil {
		return fmt.Errorf("failed to get character button element: %w", err)
	}
	defer comboElem.Release()

	ctrlType, err = comboElem.GetControlType()
	if err != nil {
		return fmt.Errorf("failed to get character button control type: %w", err)
	}
	if ctrlType != win32.UIA_ButtonControlTypeId {
		return internal.ErrElementNotFound
	}
	name, err := comboElem.GetName()
	if err != nil {
		return fmt.Errorf("failed to get character button name: %w", err)
	}
	if name == "" || name == mainWindowIconButtonName {
		return internal.ErrElementNotFound
	}

	buttonElem, err := elems.Get(comboIndex + 1)
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
	name, err = buttonElem.GetName()
	if err != nil {
		return fmt.Errorf("failed to get image button control name: %w", err)
	}
	if name != mainWindowIconButtonName {
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
		if err != nil && !errors.Is(err, internal.ErrElementNotFound) {
			log.Printf("failed to get elements: %v", err)
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
