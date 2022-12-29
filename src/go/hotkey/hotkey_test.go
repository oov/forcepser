package hotkey

import (
	"fmt"
	"testing"

	"github.com/zzl/go-win32api/win32"
)

func TestParseKey(t *testing.T) {
	tests := []struct {
		s   string
		key Key
		err error
	}{
		{
			s:   "Shift + Ctrl + C",
			key: Key{win32.MOD_SHIFT | win32.MOD_CONTROL, win32.VK_C},
		},
		{
			s:   "Shift + Ctrl2 + C",
			err: fmt.Errorf("ctrl2 is unexpected key"),
		},
	}
	for idx, data := range tests {
		key, err := ParseKey(data.s)
		if data.err == nil {
			if err != nil {
				t.Errorf("No.%d: failed: %v", idx, err)
				continue
			}
			if data.key.key != key.key {
				t.Errorf("No.%d key: want %v got %v", idx, data.key, key.key)
			}
			if data.key.mods != key.mods {
				t.Errorf("No.%d mods: want %v got %v", idx, data.key.mods, key.mods)
			}
		} else {
			if err == nil {
				t.Errorf("No.%d: should fail", idx)
				continue
			}
			if err.Error() != data.err.Error() {
				t.Errorf("No.%d: want %v got %v", idx, data.err, err)
			}
		}
	}
}

func TestString(t *testing.T) {
	tests := []string{
		"A",
		"Shift + Ctrl + B",
	}
	for idx, s := range tests {
		key, err := ParseKey(s)
		if err != nil {
			t.Errorf("No.%d failed: %v", idx, err)
		}
		str := key.String()
		if str != s {
			t.Errorf("No.%d want %v got %v", idx, s, str)
		}
	}
}
