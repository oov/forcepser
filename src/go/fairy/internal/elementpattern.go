package internal

import (
	"fmt"
	"syscall"
	"unsafe"

	"github.com/zzl/go-win32api/win32"
)

func (elem *Element) GetCurrentInvokePattern() (*win32.IUIAutomationInvokePattern, error) {
	hasInvoke, err := elem.GetCurrentPropertyInt32Value(win32.UIA_IsInvokePatternAvailablePropertyId)
	if err != nil {
		return nil, fmt.Errorf("failed to get UIA_IsInvokePatternAvailablePropertyId property")
	}
	if hasInvoke == 0 {
		return nil, fmt.Errorf("element does not have InvokePattern")
	}
	var intf *win32.IUIAutomationInvokePattern
	if hr := elem.IUIAutomationElement.GetCurrentPatternAs(win32.UIA_InvokePatternId, &win32.IID_IUIAutomationInvokePattern, unsafe.Pointer(&intf)); win32.FAILED(hr) || intf == nil {
		return nil, fmt.Errorf("IUIAutomationElement.GetCurrentPatternAs failed: %s", win32.HRESULT_ToString(hr))
	}
	return intf, nil
}

func (elem *Element) GetCurrentTogglePattern() (*win32.IUIAutomationTogglePattern, error) {
	hasInvoke, err := elem.GetCurrentPropertyInt32Value(win32.UIA_IsTogglePatternAvailablePropertyId)
	if err != nil {
		return nil, fmt.Errorf("failed to get UIA_IsTogglePatternAvailablePropertyId property")
	}
	if hasInvoke == 0 {
		return nil, fmt.Errorf("element does not have TogglePattern")
	}
	var intf *win32.IUIAutomationTogglePattern
	if hr := elem.IUIAutomationElement.GetCurrentPatternAs(win32.UIA_TogglePatternId, &win32.IID_IUIAutomationTogglePattern, unsafe.Pointer(&intf)); win32.FAILED(hr) || intf == nil {
		return nil, fmt.Errorf("IUIAutomationElement.GetCurrentPatternAs failed: %s", win32.HRESULT_ToString(hr))
	}
	return intf, nil
}

func (elem *Element) GetCurrentTextPattern() (*win32.IUIAutomationTextPattern, error) {
	has, err := elem.GetCurrentPropertyInt32Value(win32.UIA_IsTextPatternAvailablePropertyId)
	if err != nil {
		return nil, fmt.Errorf("failed to get UIA_IsTextPatternAvailablePropertyId property")
	}
	if has == 0 {
		return nil, fmt.Errorf("element does not have TextPattern")
	}
	var intf *win32.IUIAutomationTextPattern
	if hr := elem.IUIAutomationElement.GetCurrentPatternAs(win32.UIA_TextPatternId, &win32.IID_IUIAutomationTextPattern, unsafe.Pointer(&intf)); win32.FAILED(hr) || intf == nil {
		return nil, fmt.Errorf("IUIAutomationElement.GetCurrentPatternAs failed: %s", win32.HRESULT_ToString(hr))
	}
	return intf, nil
}

func (elem *Element) GetCurrentLegacyIAccessiblePattern() (*win32.IUIAutomationLegacyIAccessiblePattern, error) {
	has, err := elem.GetCurrentPropertyInt32Value(win32.UIA_IsLegacyIAccessiblePatternAvailablePropertyId)
	if err != nil {
		return nil, fmt.Errorf("failed to get UIA_IsLegacyIAccessiblePatternAvailablePropertyId property")
	}
	if has == 0 {
		return nil, fmt.Errorf("element does not have LegacyIAccessiblePattern")
	}
	var liap *win32.IUIAutomationLegacyIAccessiblePattern
	if hr := elem.GetCurrentPatternAs(win32.UIA_LegacyIAccessiblePatternId, &win32.IID_IUIAutomationLegacyIAccessiblePattern, unsafe.Pointer(&liap)); win32.FAILED(hr) || liap == nil {
		return nil, fmt.Errorf("IUIAutomationElement.GetCurrentPatternAs failed: %s", win32.HRESULT_ToString(hr))
	}
	return liap, nil
}

func (elem *Element) GetCurrentValuePattern() (*win32.IUIAutomationValuePattern, error) {
	has, err := elem.GetCurrentPropertyInt32Value(win32.UIA_IsValuePatternAvailablePropertyId)
	if err != nil {
		return nil, fmt.Errorf("failed to get UIA_IsValuePatternAvailablePropertyId property")
	}
	if has == 0 {
		return nil, fmt.Errorf("element does not have ValuePattern")
	}
	var vp *win32.IUIAutomationValuePattern
	if hr := elem.GetCurrentPatternAs(win32.UIA_ValuePatternId, &win32.IID_IUIAutomationValuePattern, unsafe.Pointer(&vp)); win32.FAILED(hr) || vp == nil {
		return nil, fmt.Errorf("IUIAutomationElement.GetCurrentPatternAs failed: %s", win32.HRESULT_ToString(hr))
	}
	return vp, nil
}

func (elem *Element) Invoke() error {
	intf, err := elem.GetCurrentInvokePattern()
	if err != nil {
		return fmt.Errorf("failed to get IUIAutomationInvokePattern: %w", err)
	}
	defer intf.Release()

	if hr := intf.Invoke(); win32.FAILED(hr) {
		return fmt.Errorf("IUIAutomationInvokePattern.Invoke failed: %s", win32.HRESULT_ToString(hr))
	}
	return nil
}

func (elem *Element) SetToggleValue(v win32.ToggleState) error {
	tp, err := elem.GetCurrentTogglePattern()
	if err != nil {
		return fmt.Errorf("failed to get IUIAutomationTogglePattern: %w", err)
	}
	defer tp.Release()

	var initial win32.ToggleState
	if hr := tp.Get_CurrentToggleState(&initial); win32.FAILED(hr) {
		return fmt.Errorf("IUIAutomationTogglePattern.Get_CurrentToggleState failed: %s", win32.HRESULT_ToString(hr))
	}
	for i := 0; i < 3; i++ {
		if hr := tp.Toggle(); win32.FAILED(hr) {
			return fmt.Errorf("IUIAutomationTogglePattern.Toggle failed: %s", win32.HRESULT_ToString(hr))
		}
		var current win32.ToggleState
		if hr := tp.Get_CurrentToggleState(&current); win32.FAILED(hr) {
			return fmt.Errorf("IUIAutomationTogglePattern.Get_CurrentToggleState failed: %s", win32.HRESULT_ToString(hr))
		}
		if current == v {
			return nil
		}
	}
	return fmt.Errorf("failed to set toggle value: %v -> %v", initial, v)
}

func (elem *Element) GetTextViaValuePattern() (string, error) {
	vp, err := elem.GetCurrentValuePattern()
	if err != nil {
		return "", fmt.Errorf("failed to get IUIAutomationValuePattern: %w", err)
	}
	defer vp.Release()
	var bs win32.BSTR
	if hr := vp.Get_CurrentValue(&bs); win32.FAILED(hr) {
		return "", fmt.Errorf("IUIAutomationValuePattern.Get_CurrentValue failed: %s", win32.HRESULT_ToString(hr))
	}
	return win32.BstrToStrAndFree(bs), nil
}

func (elem *Element) GetTextViaTextPattern() (string, error) {
	tp, err := elem.GetCurrentTextPattern()
	if err != nil {
		return "", fmt.Errorf("failed to get IUIAutomationTextPattern: %w", err)
	}
	defer tp.Release()

	var r *win32.IUIAutomationTextRange
	if hr := tp.Get_DocumentRange(&r); win32.FAILED(hr) || r == nil {
		return "", fmt.Errorf("IUIAutomationTextPattern.Get_DocumentRange failed: %s", win32.HRESULT_ToString(hr))
	}
	defer r.Release()

	var bs win32.BSTR
	if hr := r.GetText(-1, &bs); win32.FAILED(hr) {
		return "", fmt.Errorf("IUIAutomationTextRange.GetText failed: %s", win32.HRESULT_ToString(hr))
	}
	return win32.BstrToStrAndFree(bs), nil
}

func (elem *Element) SetTextViaValuePattern(text string) error {
	vp, err := elem.GetCurrentValuePattern()
	if err != nil {
		return fmt.Errorf("failed to get IUIAutomationValuePattern")
	}
	defer vp.Release()

	bs := win32.StrToBstr(text)
	defer win32.SysFreeString(bs)
	if hr := vp.SetValue(bs); win32.FAILED(hr) {
		return fmt.Errorf("IUIAutomationValuePattern.SetValue failed: %s", win32.HRESULT_ToString(hr))
	}
	return nil
}

func (elem *Element) SetTextViaLegacyIAccessiblePattern(text string) error {
	lia, err := elem.GetCurrentLegacyIAccessiblePattern()
	if err != nil {
		return fmt.Errorf("failed to get IUIAutomationLegacyIAccessiblePattern")
	}
	defer lia.Release()

	bs := win32.StrToBstr(text)
	defer win32.SysFreeString(bs)
	if hr := lia.SetValue(bs); win32.FAILED(hr) {
		return fmt.Errorf("IUIAutomationLegacyIAccessiblePattern.SetValue failed: %s", win32.HRESULT_ToString(hr))
	}
	return nil
}

func (elem *Element) SetTextViaWMCharSimple(window win32.HWND, text string) error {
	u, err := syscall.UTF16FromString(text)
	if err != nil {
		return fmt.Errorf("failed to convert string: %w", err)
	}
	for _, wc := range u[0 : len(u)-1] {
		win32.SendMessage(window, win32.WM_CHAR, win32.WPARAM(wc), 0)
	}
	return nil
}

func (elem *Element) SetTextViaWMCharSimplePost(window win32.HWND, text string) error {
	u, err := syscall.UTF16FromString(text)
	if err != nil {
		return fmt.Errorf("failed to convert string: %w", err)
	}
	for _, wc := range u[0 : len(u)-1] {
		win32.PostMessage(window, win32.WM_CHAR, win32.WPARAM(wc), 0)
	}
	return nil
}

func (elem *Element) SetTextViaWMChar(window win32.HWND, text string) error {
	// take focus
	lia, err := elem.GetCurrentLegacyIAccessiblePattern()
	if err != nil {
		return fmt.Errorf("failed to get IUIAutomationLegacyIAccessiblePattern: %w", err)
	}
	defer lia.Release()

	if hr := lia.Select(int32(win32.SELFLAG_TAKEFOCUS)); win32.FAILED(hr) {
		return fmt.Errorf("IUIAutomationLegacyIAccessiblePattern.Select failed: %s", win32.HRESULT_ToString(hr))
	}

	// select current text to overwrite
	tp, err := elem.GetCurrentTextPattern()
	if err != nil {
		return fmt.Errorf("failed to get IUIAutomationTextPattern: %w", err)
	}
	defer tp.Release()

	var tr *win32.IUIAutomationTextRange
	if hr := tp.Get_DocumentRange(&tr); win32.FAILED(hr) || tr == nil {
		return fmt.Errorf("IUIAutomationTextPattern.Get_DocumentRange failed: %s", win32.HRESULT_ToString(hr))
	}
	defer tr.Release()

	if hr := tr.Select(); win32.FAILED(hr) {
		return fmt.Errorf("IUIAutomationTextRange.Select failed: %s", win32.HRESULT_ToString(hr))
	}

	return elem.SetTextViaWMCharSimple(window, text)
}

func (elem *Element) GetFirstSelection() (*win32.IUIAutomationTextRange, error) {
	tp, err := elem.GetCurrentTextPattern()
	if err != nil {
		return nil, fmt.Errorf("failed to get IUIAutomationTextPattern: %w", err)
	}
	defer tp.Release()

	var tra *win32.IUIAutomationTextRangeArray
	if hr := tp.GetSelection(&tra); win32.FAILED(hr) {
		return nil, fmt.Errorf("IUIAutomationTextPattern.GetSelection failed: %s", win32.HRESULT_ToString(hr))
	}
	defer tra.Release()

	var n int32
	if hr := tra.Get_Length(&n); win32.FAILED(hr) {
		return nil, fmt.Errorf("IUIAutomationTextRangeArray.Get_Length failed: %s", win32.HRESULT_ToString(hr))
	}
	if n == 0 {
		return nil, nil
	}
	var tr *win32.IUIAutomationTextRange
	if hr := tra.GetElement(0, &tr); win32.FAILED(hr) {
		return nil, fmt.Errorf("IUIAutomationTextRangeArray.GetElement failed: %s", win32.HRESULT_ToString(hr))
	}
	return tr, nil
}

func (elem *Element) DoDefaultAction() error {
	lia, err := elem.GetCurrentLegacyIAccessiblePattern()
	if err != nil {
		return fmt.Errorf("failed to get IUIAutomationLegacyIAccessiblePattern: %w", err)
	}
	defer lia.Release()
	if hr := lia.DoDefaultAction(); win32.FAILED(hr) {
		return fmt.Errorf("IUIAutomationLegacyIAccessiblePattern.DoDefaultAction failed: %s", win32.HRESULT_ToString(hr))
	}
	return nil
}

func (elem *Element) ShowContextMenuViaIUIAutomationElement3() error {
	var e3 *win32.IUIAutomationElement3
	if hr := elem.IUIAutomationElement.QueryInterface(&win32.IID_IUIAutomationElement3, unsafe.Pointer(&e3)); win32.FAILED(hr) {
		return fmt.Errorf("IUnknown.QueryInterface failed: %s", win32.HRESULT_ToString(hr))
	}
	if e3 == nil {
		return fmt.Errorf("IUIAutomationElement3 not found")
	}
	defer e3.Release()
	if hr := e3.ShowContextMenu(); win32.FAILED(hr) {
		return fmt.Errorf("IUIAutomationElement3.ShowContextMenu failed: %s", win32.HRESULT_ToString(hr))
	}
	return nil
}

func (elem *Element) ShowContextMenuViaMouseClick(window win32.HWND) error {
	r, err := elem.GetCurrentPropertyRectValue(win32.UIA_BoundingRectanglePropertyId)
	if err != nil {
		return fmt.Errorf("failed to get BoundingRect: %w", err)
	}
	var pt win32.POINT
	win32.ClientToScreen(window, &pt)
	pt.X = int32(r[0]) - pt.X + 2
	pt.Y = int32(r[1]) - pt.Y + 2
	win32.PostMessage(window, win32.WM_RBUTTONDOWN, win32.WPARAM(win32.MK_RBUTTON), win32.LPARAM(win32.MAKELONG(uint16(pt.X), uint16(pt.Y))))
	win32.PostMessage(window, win32.WM_RBUTTONUP, win32.WPARAM(win32.MK_RBUTTON), win32.LPARAM(win32.MAKELONG(uint16(pt.X), uint16(pt.Y))))
	return nil
}
