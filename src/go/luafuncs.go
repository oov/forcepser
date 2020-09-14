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
				L.RaiseError("ファイル %s が開けません: %v", path, err)
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

func luaReplaceEnv(ss *setting) lua.LGFunction {
	return func(L *lua.LState) int {
		path := L.ToString(1)
		path = ss.dirReplacer.Replace(path)
		L.Push(lua.LString(path))
		return 1
	}
}

func copyFile(dst, src string) error {
	sf, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("コピー元ファイル %s を開けません: %w", src, err)
	}
	defer sf.Close()
	df, err := os.Create(dst)
	if err != nil {
		return fmt.Errorf("コピー先ファイル %s を開けません: %w", dst, err)
	}
	defer df.Close()
	_, err = io.Copy(df, sf)
	if err != nil {
		return fmt.Errorf("ファイルコピー %s -> %s に失敗しました: %w", src, dst, err)
	}
	return nil
}

func changeExt(path, ext string) string {
	return path[:len(path)-len(filepath.Ext(path))] + ext
}

func enumMoveTargetFiles(wavpath string) ([]string, error) {
	d, err := os.Open(filepath.Dir(wavpath))
	if err != nil {
		return nil, err
	}
	defer d.Close()

	fis, err := d.Readdir(0)
	if err != nil {
		return nil, err
	}

	fname := changeExt(filepath.Base(wavpath), "")
	r := []string{}
	for _, fi := range fis {
		n := fi.Name()
		if changeExt(n, "") == fname && !fi.IsDir() {
			r = append(r, n)
		}
	}
	return r, nil
}

func delayRemove(files []string, delay float64) {
	time.Sleep(time.Duration(delay) * time.Second)
	for _, f := range files {
		if err := os.Remove(f); err != nil {
			log.Printf("[WARN] 移動元のファイル %s の削除に失敗しました: %v\n", f, err)
		}
		if verbose {
			log.Println("[INFO]", "ファイル削除:", f)
		}
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
		if rule.DeleteText {
			textfile := changeExt(path, ".txt")
			err = os.Remove(textfile)
			if err != nil {
				L.RaiseError("%s が削除できません: %v", textfile, err)
			}
			log.Println("  deletetext の設定に従い txt を削除しました")
		}
		files, err := enumMoveTargetFiles(path)
		if err != nil {
			L.RaiseError("ファイルの列挙に失敗しました: %v", err)
		}
		if rule.FileMove == "move" || rule.FileMove == "copy" {
			destDir := rule.ExpandedDestDir()
			srcDir := filepath.Dir(path)
			if strings.Index(rule.DestDir, "%PROJECTDIR%") != -1 && ss.projectDir == "" {
				proj, err := readGCMZDropsData()
				if err != nil || proj.GCMZAPIVer < 1 {
					L.RaiseError("ごちゃまぜドロップス v0.3.13 以降を導入した AviUtl が見つかりません")
				}
				if proj.Width == 0 {
					L.RaiseError("`AviUtl で編集中のプロジェクトファイルが見つかりません")
				}
				L.RaiseError("AviUtl のプロジェクトファイルがまだ保存されていないため処理を続行できません")
			}
			destfi, err := getFileInfo(destDir)
			if err != nil {
				L.RaiseError("%s先フォルダー %s の情報取得に失敗しました: %v", rule.FileMove.Readable(), destDir, err)
			}
			srcfi, err := getFileInfo(srcDir)
			if err != nil {
				L.RaiseError("%s元フォルダー %s の情報取得に失敗しました: %v", rule.FileMove.Readable(), srcDir, err)
			}
			if !isSameFileInfo(destfi, srcfi) {
				deleteFiles := []string{}
				for _, f := range files {
					oldpath := filepath.Join(srcDir, f)
					newpath := filepath.Join(destDir, f)
					err = copyFile(newpath, oldpath)
					if err != nil {
						L.RaiseError("ファイルのコピーに失敗しました: %v", err)
					}
					if verbose {
						log.Println("[INFO]", "ファイルコピー", oldpath, "->", newpath)
					}
					if rule.FileMove == "move" {
						deleteFiles = append(deleteFiles, oldpath)
					}
				}
				if rule.MoveDelay > 0 {
					go delayRemove(deleteFiles, rule.MoveDelay)
				} else {
					delayRemove(deleteFiles, 0)
				}
				log.Printf("  filemove = \"%s\" の設定に従い、ファイルを以下の場所に%sしました\n", rule.FileMove, rule.FileMove.Readable())
				log.Println("    ", destDir)
				path = filepath.Join(destDir, filepath.Base(path))
			}
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
				dir := filepath.Dir(path)
				for _, f := range files {
					oldpath := filepath.Join(dir, f)
					newpath := filepath.Join(dir, changeExt(newfilename, filepath.Ext(f)))
					if err = os.Rename(oldpath, newpath); err != nil {
						L.RaiseError("ファイル名の変更に失敗しました: %v", err)
					}
					if verbose {
						log.Println("[INFO]", "ファイル名変更:", oldpath, "->", newpath)
					}
				}
				path = filepath.Join(dir, newfilename)
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

func atoich(a byte) int {
	if '0' <= a && a <= '9' {
		return int(a - '0')
	}
	return int(a&0xdf - 'A' + 10)
}

func luaFromEXOString(L *lua.LState) int {
	src := L.ToString(1)
	var buf [1024]rune
	for i := 0; i+3 < len(src); i += 4 {
		buf[i/4] = rune((atoich(src[i]) << 4) | atoich(src[i+1]) | (atoich(src[i+2]) << 12) | (atoich(src[i+3]) << 8))
		if buf[i/4] == 0 {
			L.Push(lua.LString(buf[:i/4]))
			return 1
		}
	}
	L.Push(lua.LString(buf[:]))
	return 1
}
