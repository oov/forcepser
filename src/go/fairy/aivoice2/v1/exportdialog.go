package aivoice2

import (
	"fmt"

	"github.com/oov/forcepser/fairy/internal"

	"github.com/zzl/go-win32api/win32"
)

type exportDialog struct {
	edit   *internal.Element
	export *internal.Element
}

func (ed *exportDialog) Release() {
	if ed.edit != nil {
		ed.edit.Release()
		ed.edit = nil
	}
	if ed.export != nil {
		ed.export.Release()
		ed.export = nil
	}
}

func findExportDialog(uia *internal.UIAutomation, window *internal.Element) (*exportDialog, error) {
	// find button
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
		for _, s := range exportDialogExportButtonCaptions {
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

	var parent *internal.Element
	{
		cndCtrl, err := uia.CreateInt32PropertyCondition(win32.UIA_ControlTypePropertyId, win32.UIA_GroupControlTypeId)
		if err != nil {
			return nil, fmt.Errorf("failed to create control type condition: %w", err)
		}
		defer cndCtrl.Release()

		var tw *win32.IUIAutomationTreeWalker
		if hr := uia.IUIAutomation.CreateTreeWalker(cndCtrl, &tw); win32.FAILED(hr) {
			return nil, fmt.Errorf("IUIAutomation.ControlViewWalker failed: %s", win32.HRESULT_ToString(hr))
		}
		if tw == nil {
			return nil, fmt.Errorf("IUIAutomation.ControlViewWalker failed: tw is nil")
		}
		defer tw.Release()
		var elem *win32.IUIAutomationElement
		if hr := tw.GetParentElement(export.IUIAutomationElement, &elem); win32.FAILED(hr) {
			return nil, fmt.Errorf("IUIAutomation.ControlViewWalker.GetParentElement failed: %s", win32.HRESULT_ToString(hr))
		}
		if elem == nil {
			return nil, fmt.Errorf("IUIAutomation.ControlViewWalker.GetParentElement failed: elem is nil")
		}
		parent = &internal.Element{IUIAutomationElement: elem}
		defer parent.Release()
	}

	var edit *internal.Element
	{
		cndCtrl, err := uia.CreateInt32PropertyCondition(win32.UIA_ControlTypePropertyId, win32.UIA_EditControlTypeId)
		if err != nil {
			return nil, fmt.Errorf("failed to create control type condition: %w", err)
		}
		defer cndCtrl.Release()

		edit, err = parent.FindFirst(win32.TreeScope_Children, cndCtrl)
		if err != nil {
			return nil, fmt.Errorf("failed to find first: %w", err)
		}
		defer edit.Release()
	}

	edit.AddRef()
	export.AddRef()
	return &exportDialog{
		edit:   edit,
		export: export,
	}, nil
}
