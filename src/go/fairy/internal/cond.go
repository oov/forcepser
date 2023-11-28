package internal

import (
	"fmt"

	"github.com/zzl/go-win32api/win32"
)

func (uia *UIAutomation) CreateTrueCondition() (*win32.IUIAutomationCondition, error) {
	var cond *win32.IUIAutomationCondition
	if hr := uia.IUIAutomation.CreateTrueCondition(&cond); win32.FAILED(hr) || cond == nil {
		return nil, fmt.Errorf("IUIAutomation.CreateTrueCondition failed: %s", win32.HRESULT_ToString(hr))
	}
	return cond, nil
}

func (uia *UIAutomation) CreateAndCondition(conds ...*win32.IUIAutomationCondition) (*win32.IUIAutomationCondition, error) {
	var cond *win32.IUIAutomationCondition
	if hr := uia.IUIAutomation.CreateAndConditionFromNativeArray(&conds[0], int32(len(conds)), &cond); win32.FAILED(hr) || cond == nil {
		return nil, fmt.Errorf("IUIAutomation.CreateAndConditionFromNativeArray failed: %s", win32.HRESULT_ToString(hr))
	}
	return cond, nil
}

func (uia *UIAutomation) CreateOrCondition(conds ...*win32.IUIAutomationCondition) (*win32.IUIAutomationCondition, error) {
	var cond *win32.IUIAutomationCondition
	if hr := uia.IUIAutomation.CreateOrConditionFromNativeArray(&conds[0], int32(len(conds)), &cond); win32.FAILED(hr) || cond == nil {
		return nil, fmt.Errorf("IUIAutomation.CreateOrConditionFromNativeArray failed: %s", win32.HRESULT_ToString(hr))
	}
	return cond, nil
}

func (uia *UIAutomation) CreateNotCondition(c *win32.IUIAutomationCondition) (*win32.IUIAutomationCondition, error) {
	var cond *win32.IUIAutomationCondition
	if hr := uia.IUIAutomation.CreateNotCondition(c, &cond); win32.FAILED(hr) || cond == nil {
		return nil, fmt.Errorf("IUIAutomation.CreateOrConditionFromNativeArray failed: %s", win32.HRESULT_ToString(hr))
	}
	return cond, nil
}

func (uia *UIAutomation) CreatePropertyConditionEx(propertyId int32, v *win32.VARIANT, flags win32.PropertyConditionFlags) (*win32.IUIAutomationCondition, error) {
	var cond *win32.IUIAutomationCondition
	if hr := uia.IUIAutomation.CreatePropertyConditionEx(propertyId, *v, flags, &cond); win32.FAILED(hr) || cond == nil {
		return nil, fmt.Errorf("IUIAutomation.CreatePropertyConditionEx failed: %s", win32.HRESULT_ToString(hr))
	}
	return cond, nil
}

func (uia *UIAutomation) CreateStringPropertyConditionEx(propertyId int32, val string, flags win32.PropertyConditionFlags) (*win32.IUIAutomationCondition, error) {
	v, err := strToVariant(val)
	if err != nil {
		return nil, fmt.Errorf("failed to create variant string: %w", err)
	}
	defer win32.VariantClear(v)
	return uia.CreatePropertyConditionEx(propertyId, v, flags)
}

func (uia *UIAutomation) CreateInt32PropertyCondition(propertyId int32, val int32) (*win32.IUIAutomationCondition, error) {
	var v win32.VARIANT
	v.Vt = uint16(win32.VT_I4)
	*v.IntVal() = val
	defer win32.VariantClear(&v)
	return uia.CreatePropertyConditionEx(propertyId, &v, 0)
}

func (uia *UIAutomation) CreateBoolPropertyCondition(propertyId int32, val bool) (*win32.IUIAutomationCondition, error) {
	var v win32.VARIANT
	v.Vt = uint16(win32.VT_BOOL)
	if val {
		*v.BoolVal() = win32.VARIANT_TRUE
	} else {
		*v.BoolVal() = win32.VARIANT_FALSE
	}
	defer win32.VariantClear(&v)
	return uia.CreatePropertyConditionEx(propertyId, &v, 0)
}
