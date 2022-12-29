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

func (elem *Element) GetTextViaValuePattern() (string, error) {
	tp, err := elem.GetCurrentTextPattern()
	if err != nil {
		return "", fmt.Errorf("failed to get IUIAutomationTextPattern: %w", err)
	}
	defer tp.Release()

	vp, err := elem.GetCurrentValuePattern()
	if err != nil {
		return "", fmt.Errorf("failed to get IUIAutomationValuePattern: %w", err)
	}
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

	// write
	u, err := syscall.UTF16FromString(text)
	if err != nil {
		return fmt.Errorf("failed to convert string: %w", err)
	}
	for _, wc := range u {
		win32.SendMessage(window, win32.WM_CHAR, win32.WPARAM(wc), 0)
	}
	return nil
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
