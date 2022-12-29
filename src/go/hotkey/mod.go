package hotkey

import (
	"strings"

	"github.com/zzl/go-win32api/win32"
)

var modifiers = map[string]struct {
	m   win32.HOT_KEY_MODIFIERS
	sub bool
}{
	"shift":   {m: win32.MOD_SHIFT},
	"ctrl":    {m: win32.MOD_CONTROL},
	"control": {m: win32.MOD_CONTROL, sub: true},
	"alt":     {m: win32.MOD_ALT},
	"win":     {m: win32.MOD_WIN},
	"windows": {m: win32.MOD_WIN, sub: true},
}

func strToModifier(s string) win32.HOT_KEY_MODIFIERS {
	if m, ok := modifiers[s]; ok {
		return m.m
	}
	return 0
}

func modifierToStr(m win32.HOT_KEY_MODIFIERS) string {
	var s []string
	if (m & win32.MOD_ALT) != 0 {
		s = append(s, "Alt")
	}
	if (m & win32.MOD_CONTROL) != 0 {
		s = append(s, "Ctrl")
	}
	if (m & win32.MOD_SHIFT) != 0 {
		s = append(s, "Shift")
	}
	if (m & win32.MOD_WIN) != 0 {
		s = append(s, "Win")
	}
	return strings.Join(s, " + ")
}
