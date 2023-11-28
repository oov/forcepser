package voisonatalk

import (
	"fmt"
	"strings"

	"github.com/oov/forcepser/fairy/internal"

	"github.com/zzl/go-win32api/win32"
)

type tableHeaderIndices struct {
	Enable     int
	Sentence   int
	ExportName int
}

type mainWindow struct {
	window           *internal.Element
	activeTrackName  *internal.Element
	activeSentence   *internal.Element
	activeExportName *internal.Element
	table            *internal.Element
	indices          tableHeaderIndices
}

func (mw *mainWindow) Release() {
	if mw.window != nil {
		mw.window.Release()
		mw.window = nil
	}
	if mw.activeTrackName != nil {
		mw.activeTrackName.Release()
		mw.activeTrackName = nil
	}
	if mw.activeSentence != nil {
		mw.activeSentence.Release()
		mw.activeSentence = nil
	}
	if mw.activeExportName != nil {
		mw.activeExportName.Release()
		mw.activeExportName = nil
	}
	if mw.table != nil {
		mw.table.Release()
		mw.table = nil
	}
}

func findTableHeaderIndices(uia *internal.UIAutomation, table *internal.Element) (tableHeaderIndices, error) {
	r := tableHeaderIndices{-1, -1, -1}
	cndHeader, err := uia.CreateInt32PropertyCondition(win32.UIA_ControlTypePropertyId, win32.UIA_HeaderControlTypeId)
	if err != nil {
		return r, fmt.Errorf("failed to create control type condition: %w", err)
	}
	defer cndHeader.Release()

	header, err := table.FindFirst(win32.TreeScope_Children, cndHeader)
	if err != nil {
		return r, fmt.Errorf("failed to get header: %w", err)
	}
	defer header.Release()

	headers, err := header.FindAll(win32.TreeScope_Children, cndHeader)
	if err != nil {
		return r, fmt.Errorf("failed to get header children: %w", err)
	}
	defer headers.Release()

	for i, ln := 0, headers.Len; i < ln; i++ {
		elem, err := headers.Get(i)
		if err != nil {
			return r, fmt.Errorf("failed to get header element: %w", err)
		}
		defer elem.Release()

		name, err := elem.GetName()
		if err != nil {
			return r, fmt.Errorf("failed to get header name: %w", err)
		}

		if match(name, mainWindowTableHeaderEnableCaptions) {
			r.Enable = i
		} else if match(name, mainWindowTableHeaderSentenceCaptions) {
			r.Sentence = i
		} else if match(name, mainWindowTableHeaderExportNameCaptions) {
			r.ExportName = i
		}
	}
	if r.Enable == -1 || r.Sentence == -1 || r.ExportName == -1 {
		return tableHeaderIndices{}, fmt.Errorf("required columns(Enable, Track, Sentence, ExportName) not found in table header")
	}
	return r, nil
}

func findTable(uia *internal.UIAutomation, window *internal.Element, mw *mainWindow) error {
	cndCtrl, err := uia.CreateInt32PropertyCondition(win32.UIA_ControlTypePropertyId, win32.UIA_TableControlTypeId)
	if err != nil {
		return fmt.Errorf("failed to create control type condition: %w", err)
	}
	defer cndCtrl.Release()
	table, err := window.FindFirst(win32.TreeScope_Children, cndCtrl)
	if err != nil {
		return fmt.Errorf("failed to get children: %w", err)
	}
	defer table.Release()
	indices, err := findTableHeaderIndices(uia, table)
	if err != nil {
		return fmt.Errorf("failed to find header indices: %w", err)
	}

	cndListItemCtrl, err := uia.CreateInt32PropertyCondition(win32.UIA_ControlTypePropertyId, win32.UIA_ListItemControlTypeId)
	if err != nil {
		return fmt.Errorf("failed to create list item control type condition: %w", err)
	}
	defer cndListItemCtrl.Release()
	cndSelected, err := uia.CreateBoolPropertyCondition(win32.UIA_SelectionItemIsSelectedPropertyId, true)
	if err != nil {
		return fmt.Errorf("failed to create UIA_SelectionItemIsSelectedPropertyId condition: %w", err)
	}
	defer cndSelected.Release()
	cndAnd, err := uia.CreateAndCondition(cndListItemCtrl, cndSelected)
	if err != nil {
		return fmt.Errorf("failed to create and condition: %w", err)
	}
	defer cndAnd.Release()
	activeRow, err := table.FindFirst(win32.TreeScope_Children, cndAnd)
	if err != nil {
		return fmt.Errorf("failed to get active row: %w", err)
	}
	defer activeRow.Release()

	realRow, err := activeRow.FindFirst(win32.TreeScope_Children, cndListItemCtrl)
	if err != nil {
		return fmt.Errorf("failed to get real row: %w", err)
	}
	defer realRow.Release()

	cndSentence, err := uia.CreateInt32PropertyCondition(win32.UIA_GridItemColumnPropertyId, int32(indices.Sentence))
	if err != nil {
		return fmt.Errorf("failed to create grid item column property id condition: %w", err)
	}
	defer cndSentence.Release()
	sentence, err := realRow.FindFirst(win32.TreeScope_Children, cndSentence)
	if err != nil {
		return fmt.Errorf("failed to get sentence: %w", err)
	}
	defer sentence.Release()

	cndExportName, err := uia.CreateInt32PropertyCondition(win32.UIA_GridItemColumnPropertyId, int32(indices.ExportName))
	if err != nil {
		return fmt.Errorf("failed to create grid item column property id condition: %w", err)
	}
	defer cndExportName.Release()
	exportName, err := realRow.FindFirst(win32.TreeScope_Children, cndExportName)
	if err != nil {
		return fmt.Errorf("failed to get exportName: %w", err)
	}
	defer exportName.Release()

	table.AddRef()
	sentence.AddRef()
	exportName.AddRef()
	mw.table = table
	mw.indices = indices
	mw.activeSentence = sentence
	mw.activeExportName = exportName
	return nil
}

func findActiveTrackName(uia *internal.UIAutomation, window *internal.Element, mw *mainWindow) error {
	cndListCtrl, err := uia.CreateInt32PropertyCondition(win32.UIA_ControlTypePropertyId, win32.UIA_ListControlTypeId)
	if err != nil {
		return fmt.Errorf("failed to create control type condition: %w", err)
	}
	defer cndListCtrl.Release()
	list, err := window.FindFirst(win32.TreeScope_Children, cndListCtrl)
	if err != nil {
		return fmt.Errorf("failed to get list: %w", err)
	}
	defer list.Release()

	cndListItemCtrl, err := uia.CreateInt32PropertyCondition(win32.UIA_ControlTypePropertyId, win32.UIA_ListItemControlTypeId)
	if err != nil {
		return fmt.Errorf("failed to create list item control type condition: %w", err)
	}
	defer cndListItemCtrl.Release()
	cndSelected, err := uia.CreateBoolPropertyCondition(win32.UIA_SelectionItemIsSelectedPropertyId, true)
	if err != nil {
		return fmt.Errorf("failed to create UIA_SelectionItemIsSelectedPropertyId condition: %w", err)
	}
	defer cndSelected.Release()
	cndAnd, err := uia.CreateAndCondition(cndListItemCtrl, cndSelected)
	if err != nil {
		return fmt.Errorf("failed to create and condition: %w", err)
	}
	defer cndAnd.Release()
	item, err := list.FindFirst(win32.TreeScope_Children, cndAnd)
	if err != nil {
		return fmt.Errorf("failed to get list item: %w", err)
	}
	defer item.Release()

	cndEdit, err := uia.CreateInt32PropertyCondition(win32.UIA_ControlTypePropertyId, win32.UIA_EditControlTypeId)
	if err != nil {
		return fmt.Errorf("failed to create edit control type condition: %w", err)
	}
	defer cndEdit.Release()
	edit, err := item.FindFirst(win32.TreeScope_Children, cndEdit)
	if err != nil {
		return fmt.Errorf("failed to get edit: %w", err)
	}
	mw.activeTrackName = edit
	return nil
}

func (mw *mainWindow) updateCheckBoxes(uia *internal.UIAutomation) error {
	cndCtrl, err := uia.CreateInt32PropertyCondition(win32.UIA_ControlTypePropertyId, win32.UIA_ListItemControlTypeId)
	if err != nil {
		return fmt.Errorf("failed to create list item control type condition: %w", err)
	}
	defer cndCtrl.Release()
	cndColumn, err := uia.CreateInt32PropertyCondition(win32.UIA_GridItemColumnPropertyId, int32(mw.indices.Enable))
	if err != nil {
		return fmt.Errorf("failed to create grid item column property id condition: %w", err)
	}
	defer cndColumn.Release()
	cndCheckBox, err := uia.CreateInt32PropertyCondition(win32.UIA_ControlTypePropertyId, win32.UIA_CheckBoxControlTypeId)
	if err != nil {
		return fmt.Errorf("failed to create control type condition: %w", err)
	}
	defer cndCheckBox.Release()

	rows, err := mw.table.FindAll(win32.TreeScope_Children, cndCtrl)
	if err != nil {
		return fmt.Errorf("failed to get rows: %w", err)
	}
	defer rows.Release()

	for i, ln := 0, rows.Len; i < ln; i++ {
		err = func() error {
			row, err := rows.Get(i)
			if err != nil {
				return fmt.Errorf("failed to get row element: %w", err)
			}
			defer row.Release()

			isactive, err := row.GetCurrentPropertyInt32Value(win32.UIA_SelectionItemIsSelectedPropertyId)
			if err != nil {
				return fmt.Errorf("failed to get isactive value: %w", err)
			}

			realRow, err := row.FindFirst(win32.TreeScope_Children, cndCtrl)
			if err != nil {
				return fmt.Errorf("failed to get real row: %w", err)
			}
			defer realRow.Release()

			checkBoxContainer, err := realRow.FindFirst(win32.TreeScope_Children, cndColumn)
			if err != nil {
				return fmt.Errorf("failed to get check box container: %w", err)
			}
			defer checkBoxContainer.Release()

			checkBox, err := checkBoxContainer.FindFirst(win32.TreeScope_Children, cndCheckBox)
			if err != nil {
				if err == internal.ErrElementNotFound && isactive == 0 {
					// not found but not active, can be ignored.
					// occurs when the sentence is empty.
					return nil
				}
				return fmt.Errorf("failed to get checkbox: %w", err)
			}
			defer checkBox.Release()

			if isactive == 0 {
				err = checkBox.SetToggleValue(win32.ToggleState_Off)
			} else {
				err = checkBox.SetToggleValue(win32.ToggleState_On)
			}
			if err != nil {
				return fmt.Errorf("failed to set toggle value: %w", err)
			}
			return nil
		}()
		if err != nil {
			return fmt.Errorf("failed to update checkbox: %w", err)
		}
	}
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

	name, err := window.GetName()
	if err != nil {
		return nil, fmt.Errorf("failed to get window name: %w", err)
	}
	if !strings.Contains(name, mainWindowName) {
		return nil, fmt.Errorf("window name mismatch: %q", name)
	}

	mw := mainWindow{window: window}
	err = findActiveTrackName(uia, window, &mw)
	if err != nil {
		return nil, fmt.Errorf("failed to find active track name: %w", err)
	}
	defer mw.activeTrackName.Release()
	err = findTable(uia, window, &mw)
	if err != nil {
		return nil, fmt.Errorf("failed to find table: %w", err)
	}
	defer mw.activeSentence.Release()
	defer mw.activeExportName.Release()

	window.AddRef()
	mw.activeExportName.AddRef()
	mw.activeSentence.AddRef()
	mw.activeTrackName.AddRef()
	return &mw, nil
}
