package hotkey

import (
	"strings"

	"github.com/zzl/go-win32api/win32"
)

var keys = map[string]struct {
	k   win32.VIRTUAL_KEY
	sub bool
}{
	"0": {k: win32.VK_0},
	"1": {k: win32.VK_1},
	"2": {k: win32.VK_2},
	"3": {k: win32.VK_3},
	"4": {k: win32.VK_4},
	"5": {k: win32.VK_5},
	"6": {k: win32.VK_6},
	"7": {k: win32.VK_7},
	"8": {k: win32.VK_8},
	"9": {k: win32.VK_9},
	"a": {k: win32.VK_A},
	"b": {k: win32.VK_B},
	"c": {k: win32.VK_C},
	"d": {k: win32.VK_D},
	"e": {k: win32.VK_E},
	"f": {k: win32.VK_F},
	"g": {k: win32.VK_G},
	"h": {k: win32.VK_H},
	"i": {k: win32.VK_I},
	"j": {k: win32.VK_J},
	"k": {k: win32.VK_K},
	"l": {k: win32.VK_L},
	"m": {k: win32.VK_M},
	"n": {k: win32.VK_N},
	"o": {k: win32.VK_O},
	"p": {k: win32.VK_P},
	"q": {k: win32.VK_Q},
	"r": {k: win32.VK_R},
	"s": {k: win32.VK_S},
	"t": {k: win32.VK_T},
	"u": {k: win32.VK_U},
	"v": {k: win32.VK_V},
	"w": {k: win32.VK_W},
	"x": {k: win32.VK_X},
	"y": {k: win32.VK_Y},
	"z": {k: win32.VK_Z},

	"left":  {k: win32.VK_LEFT},
	"right": {k: win32.VK_RIGHT},
	"up":    {k: win32.VK_UP},
	"down":  {k: win32.VK_DOWN},

	"space":  {k: win32.VK_SPACE},
	"return": {k: win32.VK_RETURN},
	"enter":  {k: win32.VK_RETURN, sub: true},
	"esc":    {k: win32.VK_ESCAPE},
	"escape": {k: win32.VK_ESCAPE, sub: true},
	"del":    {k: win32.VK_DELETE},
	"delete": {k: win32.VK_DELETE, sub: true},
	"tab":    {k: win32.VK_TAB},

	"f1":  {k: win32.VK_F1},
	"f2":  {k: win32.VK_F2},
	"f3":  {k: win32.VK_F3},
	"f4":  {k: win32.VK_F4},
	"f5":  {k: win32.VK_F5},
	"f6":  {k: win32.VK_F6},
	"f7":  {k: win32.VK_F7},
	"f8":  {k: win32.VK_F8},
	"f9":  {k: win32.VK_F9},
	"f10": {k: win32.VK_F10},
	"f11": {k: win32.VK_F11},
	"f12": {k: win32.VK_F12},
	"f13": {k: win32.VK_F13},
	"f14": {k: win32.VK_F14},
	"f15": {k: win32.VK_F15},
	"f16": {k: win32.VK_F16},
	"f17": {k: win32.VK_F17},
	"f18": {k: win32.VK_F18},
	"f19": {k: win32.VK_F19},
	"f20": {k: win32.VK_F20},
}

func strToKey(s string) win32.VIRTUAL_KEY {
	if v, ok := keys[s]; ok {
		return v.k
	}
	return 0
}

func keyToStr(k win32.VIRTUAL_KEY) string {
	for kk, v := range keys {
		if k == v.k && !v.sub {
			return strings.ToUpper(kk[0:1]) + kk[1:]
		}
	}
	return ""
}
