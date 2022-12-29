package internal

import (
	"fmt"

	"github.com/zzl/go-win32api/win32"
)

type Elements struct {
	*win32.IUIAutomationElementArray
	Len int
}

func (elems *Elements) Get(i int) (*Element, error) {
	var elem *win32.IUIAutomationElement
	if hr := elems.IUIAutomationElementArray.GetElement(int32(i), &elem); win32.FAILED(hr) {
		return nil, fmt.Errorf("IUIAutomationElementArray.GetElement failed: %s", win32.HRESULT_ToString(hr))
	}
	return &Element{elem}, nil
}
