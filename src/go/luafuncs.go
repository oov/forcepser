package main

import (
	"log"
	"os"
	"unicode/utf16"

	"github.com/oov/audio/wave"
	"github.com/yuin/gluare"
	lua "github.com/yuin/gopher-lua"
	"golang.org/x/text/encoding/japanese"
)

func luaDebugPrint(L *lua.LState) int {
	log.Println(L.ToString(1))
	return 0
}

func luaFindRule(ss *setting) lua.LGFunction {
	return func(L *lua.LState) int {
		rule, text, err := ss.Find(L.ToString(1))
		if err != nil {
			L.RaiseError("マッチ条件の検索中にエラーが発生しました: %v", err)
		}
		if rule == nil {
			return 0
		}

		if rule.Modifier != "" {
			L2 := lua.NewState()
			defer L2.Close()
			L2.PreloadModule("re", gluare.Loader)
			if err = L2.DoString(`re = require("re")`); err != nil {
				L.RaiseError("modifier スクリプトの初期化中にエラーが発生しました: %v", err)
			}
			L2.SetGlobal("text", lua.LString(text))
			if err := L2.DoString(rule.Modifier); err != nil {
				L.RaiseError("modifier スクリプトの実行中にエラーが発生しました: %v", err)
			}
			text = L2.GetGlobal("text").String()
		}

		t := L.NewTable()
		t.RawSetString("dir", lua.LString(rule.Dir))
		t.RawSetString("file", lua.LString(rule.File))
		t.RawSetString("encoding", lua.LString(rule.Encoding))
		t.RawSetString("text", lua.LString(rule.Text))
		t.RawSetString("layer", lua.LNumber(rule.Layer))
		L.Push(t)
		L.Push(lua.LString(text))
		return 2
	}
}

func luaGetAudioInfo(L *lua.LState) int {
	f, err := os.Open(L.ToString(1))
	if err != nil {
		return 0
	}
	defer f.Close()
	r, wfe, err := wave.NewLimitedReader(f)
	if err != nil {
		return 0
	}
	t := L.NewTable()
	t.RawSetString("samplerate", lua.LNumber(wfe.Format.SamplesPerSec))
	t.RawSetString("channels", lua.LNumber(wfe.Format.Channels))
	t.RawSetString("bits", lua.LNumber(wfe.Format.BitsPerSample))
	t.RawSetString("samples", lua.LNumber(r.N/int64(wfe.Format.Channels)/int64(wfe.Format.BitsPerSample/8)))
	L.Push(t)
	return 1
}

func luaToSJIS(L *lua.LState) int {
	s, err := japanese.ShiftJIS.NewEncoder().String(L.ToString(1))
	if err != nil {
		return 0
	}
	L.Push(lua.LString(s))
	return 1
}

var hexChars = [16]byte{'0', '1', '2', '3', '4', '5', '6', '7', '8', '9', 'a', 'b', 'c', 'd', 'e', 'f'}

func luaToEXOString(L *lua.LState) int {
	u16 := utf16.Encode([]rune(L.ToString(1)))
	buf := make([]byte, 1024*4)
	for i, c := range u16 {
		buf[i*4+0] = hexChars[(c>>4)&15]
		buf[i*4+1] = hexChars[(c>>0)&15]
		buf[i*4+2] = hexChars[(c>>12)&15]
		buf[i*4+3] = hexChars[(c>>8)&15]
	}
	for i := len(u16) * 4; i < len(buf); i++ {
		buf[i] = '0'
	}
	L.Push(lua.LString(buf))
	return 1
}
