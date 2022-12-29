package internal

import (
	"fmt"
	"syscall"

	"github.com/zzl/go-win32api/win32"
)

func strToVariant(s string) (*win32.VARIANT, error) {
	ws, err := syscall.UTF16PtrFromString(s)
	if err != nil {
		return nil, fmt.Errorf("failed to convert utf-8 to utf-16: %w", err)
	}
	bstr := win32.SysAllocString(ws)
	if bstr == nil {
		return nil, fmt.Errorf("failed to create string")
	}
	var v win32.VARIANT
	v.Vt = uint16(win32.VT_BSTR)
	*v.BstrVal() = bstr
	return &v, nil
}

func variantToInt32(v *win32.VARIANT) (int32, error) {
	if v.Vt == uint16(win32.VT_I4) {
		return v.IntValVal(), nil
	}
	var tmp win32.VARIANT
	defer win32.VariantClear(&tmp)
	if hr := win32.VariantChangeType(&tmp, v, 0, uint16(win32.VT_I4)); win32.FAILED(hr) {
		return 0, fmt.Errorf("VariantChangeType failed: %s", win32.HRESULT_ToString(hr))
	}
	return tmp.IntValVal(), nil
}
