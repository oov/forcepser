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

func copyFile(dst, src string) error {
	sf, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("コピー元ファイル %q を開けません: %w", src, err)
	}
	defer sf.Close()
	df, err := os.Create(dst)
	if err != nil {
		return fmt.Errorf("コピー先ファイル %q を開けません: %w", dst, err)
	}
	defer df.Close()
	_, err = io.Copy(df, sf)
	if err != nil {
		return fmt.Errorf("ファイルコピー %q -> %q に失敗しました: %w", src, dst, err)
	}
	return nil
}

func changeExt(path, ext string) string {
	return path[:len(path)-len(filepath.Ext(path))] + ext
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
		if rule.FileMove == "move" || rule.FileMove == "copy" {
			proj := L.ToTable(2)
			if proj == nil {
				L.RaiseError("findrule でプロジェクトデータが渡されていません")
			}
			if lua.LVAsNumber(proj.RawGetString("gcmzapiver")) < 1 {
				L.RaiseError("`filemove = %q` を使うためには ごちゃまぜドロップス v0.3.13 以降が必要です", ss.FileMove)
			}
			projfile := lua.LVAsString(proj.RawGetString("projectfile"))
			if projfile == "" {
				L.RaiseError("`filemove = %q` を使うためには AviUtl のプロジェクトファイルを保存してください", ss.FileMove)
			}
			newpath := filepath.Join(filepath.Dir(projfile), filepath.Base(path))
			err = copyFile(newpath, path)
			if err != nil {
				L.RaiseError("ファイルのコピーに失敗しました: %v", err)
			}
			if verbose {
				log.Println("[INFO]", "ファイルコピー", path, "->", newpath)
			}
			txtpath, newtxtpath := changeExt(path, ".txt"), changeExt(newpath, ".txt")
			err = copyFile(newtxtpath, txtpath)
			if err != nil {
				L.RaiseError("ファイルのコピーに失敗しました: %v", err)
			}
			if verbose {
				log.Println("[INFO]", "ファイルコピー", txtpath, "->", newtxtpath)
			}
			switch ss.FileMove {
			case "copy":
				log.Println("  filemove の設定に従い wav と txt をプロジェクトファイルと同じ場所にコピーしました")
			case "move":
				err = os.Remove(path)
				if err != nil {
					L.RaiseError("移動元のファイル %q が削除できません: %v", path, err)
				}
				if verbose {
					log.Println("[INFO]", "ファイル削除:", path)
				}
				err = os.Remove(txtpath)
				if err != nil {
					L.RaiseError("移動元のファイル %q が削除できません: %v", txtpath, err)
				}
				if verbose {
					log.Println("[INFO]", "ファイル削除:", txtpath)
				}
				log.Println("  filemove の設定に従い wav と txt をプロジェクトファイルと同じ場所に移動しました")
			}
			path = newpath
		}
		if rule.DeleteText {
			textfile := changeExt(path, ".txt")
			err = os.Remove(textfile)
			if err != nil {
				L.RaiseError("%q が削除できません: %v", textfile, err)
			}
			log.Println("  deletetext の設定に従い txt を削除しました")
		}
		layer := rule.Layer
		padding := lua.LValue(lua.LNumber(rule.Padding))
		userdata := lua.LValue(lua.LString(rule.UserData))
		exofile := lua.LValue(lua.LString(rule.ExoFile))
		luafile := lua.LValue(lua.LString(rule.LuaFile))
		if rule.Modifier != "" {
			L2 := lua.NewState()
			defer L2.Close()
			L2.PreloadModule("re", gluare.Loader)
			if err = L2.DoString(`re = require("re")`); err != nil {
				L.RaiseError("modifier スクリプトの初期化中にエラーが発生しました: %v", err)
			}
			L2.SetGlobal("debug_print", L2.NewFunction(luaDebugPrint))
			L2.SetGlobal("debug_print_verbose", L2.NewFunction(luaDebugPrintVerbose))
			L2.SetGlobal("getaudioinfo", L2.NewFunction(luaGetAudioInfo))
			L2.SetGlobal("execute", L2.NewFunction(luaExecute(path, text)))
			L2.SetGlobal("tofilename", L2.NewFunction(luaToFilename))
			L2.SetGlobal("layer", lua.LNumber(layer))
			L2.SetGlobal("text", lua.LString(text))
			filename := filepath.Base(path)
			L2.SetGlobal("filename", lua.LString(filename))
			L2.SetGlobal("wave", lua.LString(path))
			L2.SetGlobal("padding", padding)
			L2.SetGlobal("userdata", userdata)
			L2.SetGlobal("exofile", exofile)
			L2.SetGlobal("luafile", luafile)
			if err = L2.DoString(rule.Modifier); err != nil {
				L.RaiseError("modifier スクリプトの実行中にエラーが発生しました: %v", err)
			}
			layer = int(lua.LVAsNumber(L2.GetGlobal("layer")))
			text = L2.GetGlobal("text").String()
			padding = L2.GetGlobal("padding")
			userdata = L2.GetGlobal("userdata")
			exofile = L2.GetGlobal("exofile")
			luafile = L2.GetGlobal("luafile")

			if newfilename := L2.GetGlobal("filename").String(); filename != newfilename {
				newpath := filepath.Join(filepath.Dir(path), newfilename)
				if err = os.Rename(path, newpath); err != nil {
					L.RaiseError("ファイル名の変更に失敗しました: %v", err)
				}
				if !ss.DeleteText {
					if err = os.Rename(changeExt(path, ".txt"), changeExt(newpath, ".txt")); err != nil {
						L.RaiseError("ファイル名の変更に失敗しました: %v", err)
					}
				}
				path = newpath
			}
		}

		t := L.NewTable()
		t.RawSetString("dir", lua.LString(rule.Dir))
		t.RawSetString("file", lua.LString(rule.File))
		t.RawSetString("encoding", lua.LString(rule.Encoding))
		t.RawSetString("layer", lua.LNumber(layer))
		t.RawSetString("text", lua.LString(rule.Text))
		t.RawSetString("userdata", userdata)
		t.RawSetString("padding", padding)
		t.RawSetString("exofile", exofile)
		t.RawSetString("luafile", luafile)
		L.Push(t)
		L.Push(lua.LString(text))
		L.Push(lua.LString(path))
		return 3
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

func luaFromSJIS(L *lua.LState) int {
	s, err := japanese.ShiftJIS.NewDecoder().String(L.ToString(1))
	if err != nil {
		L.RaiseError("文字列を Shift_JIS から変換できません: %v", err)
	}
	L.Push(lua.LString(s))
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

func luaToFilename(L *lua.LState) int {
	var nc int
	var rs []rune
	n := int(L.ToNumber(2))
	for _, c := range L.ToString(1) {
		switch c {
		case
			0x00, 0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07,
			0x08, 0x09, 0x0a, 0x0b, 0x0c, 0x0d, 0x0e, 0x0f,
			0x10, 0x11, 0x12, 0x13, 0x14, 0x15, 0x16, 0x17,
			0x18, 0x19, 0x1a, 0x1b, 0x1c, 0x1d, 0x1e, 0x1f,
			0x20, 0x22, 0x2a, 0x2f, 0x3a, 0x3c, 0x3e, 0x3f, 0x7c, 0x7f:
			continue
		}
		nc++
		if nc == n+1 {
			rs[len(rs)-1] = '…'
			break
		}
		rs = append(rs, c)
	}
	L.Push(lua.LString(string(rs)))
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
