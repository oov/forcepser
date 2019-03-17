package main

import (
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
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

func luaDebugPrintVerbose(L *lua.LState) int {
	if verbose {
		log.Println("[INFO]", L.ToString(1))
	}
	return 0
}

func luaExecute(path string, text string) lua.LGFunction {
	return func(L *lua.LState) int {
		nargs := L.GetTop()
		if nargs == 0 {
			return 0
		}
		tempFile := filepath.Join(os.TempDir(), fmt.Sprintf("forcepser%d.wav", time.Now().UnixNano()))
		defer os.Remove(tempFile)
		replacer := strings.NewReplacer("%BEFORE%", path, "%AFTER%", tempFile)
		var cmds []string
		for i := 1; i < nargs+1; i++ {
			cmds = append(cmds, replacer.Replace(L.ToString(i)))
		}
		if err := exec.Command(cmds[0], cmds[1:]...).Run(); err != nil {
			L.RaiseError("外部コマンド実行に失敗しました: %v", err)
		}
		f, err := os.Open(tempFile)
		if err == nil {
			defer f.Close()
			f2, err := os.Create(path)
			if err != nil {
				L.RaiseError("ファイル %q が開けません: %v", path, err)
			}
			defer f2.Close()
			_, err = io.Copy(f2, f)
			if err != nil {
				L.RaiseError("ファイルのコピー中にエラーが発生しました: %v", err)
			}
		}
		return 0
	}
}

func luaFindRule(ss *setting) lua.LGFunction {
	return func(L *lua.LState) int {
		path := L.ToString(1)
		rule, text, err := ss.Find(path)
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
			L2.SetGlobal("debug_print", L.NewFunction(luaDebugPrint))
			L2.SetGlobal("debug_print_verbose", L.NewFunction(luaDebugPrintVerbose))
			L2.SetGlobal("execute", L.NewFunction(luaExecute(path, text)))
			L2.SetGlobal("text", lua.LString(text))
			L2.SetGlobal("wave", lua.LString(path))
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
		L.RaiseError("ファイルが開けません: %v", err)
	}
	defer f.Close()
	r, wfe, err := wave.NewLimitedReader(f)
	if err != nil {
		L.RaiseError("Wave ファイルの読み取りに失敗しました: %v", err)
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
		L.RaiseError("文字列を Shift_JIS に変換できません: %v", err)
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
