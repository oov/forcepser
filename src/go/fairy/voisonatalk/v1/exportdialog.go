package voisonatalk

import (
	"errors"
	"fmt"
	"log"

	"github.com/oov/forcepser/fairy/internal"

	"github.com/zzl/go-win32api/win32"
)

type exportDialog struct {
	window   *internal.Element
	checkBox *internal.Element
	button   *internal.Element
}

func (ed *exportDialog) Release() {
	if ed.window != nil {
		ed.window.SetEnable(true)
		ed.window.Release()
		ed.window = nil
	}
	if ed.checkBox != nil {
		ed.checkBox.Release()
		ed.checkBox = nil
	}
	if ed.button != nil {
		ed.button.Release()
		ed.button = nil
	}
}

func findExportCheckBox(elems *internal.Elements, index int, out *exportDialog) error {
	if index < 0 || index >= elems.Len {
		return internal.ErrElementNotFound
	}
	elem, err := elems.Get(index)
	if err != nil {
		return fmt.Errorf("failed to get naming rule checkbox element: %w", err)
	}
	defer elem.Release()

	ctrlType, err := elem.GetControlType()
	if err != nil {
		return fmt.Errorf("failed to get naming rule checkbox control type: %w", err)
	}
	if ctrlType != win32.UIA_CheckBoxControlTypeId {
		return internal.ErrElementNotFound
	}
	name, err := elem.GetName()
	if err != nil {
		return fmt.Errorf("failed to get naming rule checkbox caption: %w", err)
	}
	if !match(name, exportDialogCheckBoxLabels) {
		return internal.ErrElementNotFound
	}

	elem.AddRef()
	out.checkBox = elem
	return nil
}

func findExportDialog(uia *internal.UIAutomation, pid win32.DWORD, mainWindow win32.HWND) (*exportDialog, error) {
	var nameConds []*win32.IUIAutomationCondition
	for _, s := range exportDialogTitles {
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

	window, err := findSubWindow(uia, pid, mainWindow, orCond)
	if err != nil {
		return nil, fmt.Errorf("block export dialog not found: %w", err)
	}
	defer window.Release()

	cndTrue, err := uia.CreateTrueCondition()
	if err != nil {
		return nil, fmt.Errorf("failed to create true condition: %w", err)
	}
	defer cndTrue.Release()

	elems, err := window.FindAll(win32.TreeScope_Children, cndTrue)
	if err != nil {
		return nil, fmt.Errorf("FindAll failed: %w", err)
	}
	defer elems.Release()

	exportButtonElement, err := elems.Get(elems.Len - 1)
	if err != nil {
		return nil, fmt.Errorf("failed to get export button element: %w", err)
	}
	defer exportButtonElement.Release()

	ctrlType, err := exportButtonElement.GetControlType()
	if err != nil {
		return nil, fmt.Errorf("failed to get export button control type: %w", err)
	}
	if ctrlType != win32.UIA_ButtonControlTypeId {
		return nil, fmt.Errorf("export button not found: %w", err)
	}
	name, err := exportButtonElement.GetCurrentPropertyStringValue(win32.UIA_NamePropertyId)
	if err != nil {
		return nil, fmt.Errorf("failed to get export button caption: %w", err)
	}
	if !match(name, exportDialogButtonCaptions) {
		return nil, fmt.Errorf("unexpected export button caption: %q", name)
	}
	r := exportDialog{
		window: window,
		button: exportButtonElement,
	}
	for i := 0; i < elems.Len && r.checkBox == nil; i++ {
		err = findExportCheckBox(elems, i, &r)
		if err != nil && !errors.Is(err, internal.ErrElementNotFound) {
			log.Printf("failed to get check box element: %v", err)
		}
	}
	if r.checkBox == nil {
		return nil, fmt.Errorf("check box not found: %w", err)
	}
	defer r.checkBox.Release()

	err = window.SetEnable(false)
	if err != nil {
		return nil, fmt.Errorf("failed to disable window")
	}

	window.AddRef()
	exportButtonElement.AddRef()
	r.checkBox.AddRef()
	return &r, nil
}
