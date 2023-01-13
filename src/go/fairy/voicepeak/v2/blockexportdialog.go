package voicepeak

import (
	"errors"
	"fmt"
	"log"

	"github.com/oov/forcepser/fairy/internal"

	"github.com/zzl/go-win32api/win32"
)

type blockExportDialog struct {
	window             *internal.Element
	edit               *internal.Element
	namingRuleCheckBox *internal.Element
	button             *internal.Element
}

func (bed *blockExportDialog) Release() {
	if bed.window != nil {
		bed.window.SetEnable(true)
		bed.window.Release()
		bed.window = nil
	}
	if bed.edit != nil {
		bed.edit.Release()
		bed.edit = nil
	}
	if bed.namingRuleCheckBox != nil {
		bed.namingRuleCheckBox.Release()
		bed.namingRuleCheckBox = nil
	}
	if bed.button != nil {
		bed.button.Release()
		bed.button = nil
	}
}

func findBlockExportFilenameEdit(elems *internal.Elements, index int, out *blockExportDialog) error {
	if index < 1 || index >= elems.Len {
		return internal.ErrElementNotFound
	}
	elem, err := elems.Get(index)
	if err != nil {
		return fmt.Errorf("failed to get filename edit element: %w", err)
	}
	defer elem.Release()

	ctrlType, err := elem.GetControlType()
	if err != nil {
		return fmt.Errorf("failed to get filename edit control type: %w", err)
	}
	if ctrlType != win32.UIA_EditControlTypeId {
		return internal.ErrElementNotFound
	}

	labelElem, err := elems.Get(index - 1)
	if err != nil {
		return fmt.Errorf("failed to get filename edit label element: %w", err)
	}
	defer labelElem.Release()

	ctrlType, err = labelElem.GetControlType()
	if err != nil {
		return fmt.Errorf("failed to get filename edit label control type: %w", err)
	}
	if ctrlType != win32.UIA_TextControlTypeId {
		return internal.ErrElementNotFound
	}
	name, err := labelElem.GetName()
	if err != nil {
		return fmt.Errorf("failed to get filename edit label caption: %w", err)
	}
	if !match(name, exportDialogFilenameLabels) {
		return internal.ErrElementNotFound
	}

	elem.AddRef()
	out.edit = elem
	return nil
}

func findBlockExportNamingRuleCheckBox(elems *internal.Elements, index int, out *blockExportDialog) error {
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
	if !match(name, exportDialogNamingRuleLabels) {
		return internal.ErrElementNotFound
	}

	elem.AddRef()
	out.namingRuleCheckBox = elem
	return nil
}

func findBlockExportDialog(uia *internal.UIAutomation, pid win32.DWORD, mainWindow win32.HWND) (*blockExportDialog, error) {
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

	exportButtonElement, err := elems.Get(elems.Len - 2)
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
	r := blockExportDialog{
		window: window,
		button: exportButtonElement,
	}
	for i := 0; i < elems.Len && (r.edit == nil || r.namingRuleCheckBox == nil); i++ {
		err = findBlockExportFilenameEdit(elems, i, &r)
		if err != nil && !errors.Is(err, internal.ErrElementNotFound) {
			log.Printf("failed to get edit element: %v", err)
		}
		err = findBlockExportNamingRuleCheckBox(elems, i, &r)
		if err != nil && !errors.Is(err, internal.ErrElementNotFound) {
			log.Printf("failed to get naming rule check box element: %v", err)
		}
	}
	if r.edit == nil {
		return nil, fmt.Errorf("filename edit not found: %w", err)
	}
	defer r.edit.Release()
	if r.namingRuleCheckBox == nil {
		return nil, fmt.Errorf("naming rule check box not found: %w", err)
	}
	defer r.namingRuleCheckBox.Release()

	err = window.SetEnable(false)
	if err != nil {
		return nil, fmt.Errorf("failed to disable window")
	}

	window.AddRef()
	exportButtonElement.AddRef()
	r.edit.AddRef()
	r.namingRuleCheckBox.AddRef()
	return &r, nil
}
