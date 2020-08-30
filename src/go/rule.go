package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	toml "github.com/pelletier/go-toml"
	"golang.org/x/text/encoding/japanese"
	"golang.org/x/text/encoding/unicode"
)

type rule struct {
	Dir      string
	File     string
	Encoding string
	Layer    int
	Modifier string
	Text     string
	UserData string

	ExoFile    string
	LuaFile    string
	FileMove   string
	DeleteText bool
	Padding    int

	fileRE *regexp.Regexp
	textRE *regexp.Regexp
}

func (r *rule) ExistsDir() bool {
	_, err := os.Stat(r.Dir)
	return err == nil
}

type setting struct {
	BaseDir    string
	FileMove   string
	DeleteText bool
	Delta      float64
	Freshness  float64
	ExoFile    string
	LuaFile    string
	Padding    int
	Rule       []rule
	Asas       []asas

	tempDir    string
	projectDir string
}

func makeWildcard(s string) (*regexp.Regexp, error) {
	buf := make([]byte, 0, 64)
	buf = append(buf, '^')
	pos := 0
	for i, c := range []byte(s) {
		if c != '*' && c != '?' {
			continue
		}
		if i != pos {
			buf = append(buf, regexp.QuoteMeta(s[pos:i])...)
		}
		switch c {
		case '*':
			buf = append(buf, `[^/\\]*?`...)
		case '?':
			buf = append(buf, `[^/\\]`...)
		}
		pos = i + 1
	}
	if pos != len(s) {
		buf = append(buf, regexp.QuoteMeta(s[pos:])...)
	}
	buf = append(buf, '$')
	return regexp.Compile(string(buf))
}

func newSetting(path string, tempDir string, projectDir string) (*setting, error) {
	config, err := loadTOML(path)
	if err != nil {
		return nil, fmt.Errorf("could not read setting file: %w", err)
	}
	var s setting
	s.tempDir = tempDir
	s.projectDir = projectDir
	s.BaseDir = getString("basedir", config, "")

	s.Delta = getFloat64("delta", config, 15.0)
	s.Freshness = getFloat64("freshness", config, 5.0)
	s.Padding = getInt("padding", config, 0)
	s.ExoFile = getString("exofile", config, "template.exo")
	s.LuaFile = getString("luafile", config, "genexo.lua")

	s.FileMove = getString("filemove", config, "off")
	switch s.FileMove {
	case "off", "copy", "move":
		break
	default:
		s.FileMove = "off"
	}
	s.DeleteText = getBool("deletetext", config, false)

	dirReplacer := strings.NewReplacer("%BASEDIR%", s.BaseDir, "%TEMPDIR%", s.tempDir, "%PROJECTDIR%", s.projectDir)

	for _, tr := range getSubTreeArray("rule", config) {
		var r rule
		r.Dir = getString("dir", tr, "%TEMPDIR%")
		r.Dir = dirReplacer.Replace(r.Dir)

		r.Encoding = getString("encoding", tr, "sjis")

		r.Layer = getInt("layer", tr, 1)

		r.File = getString("file", tr, "*.wav")
		r.fileRE, err = makeWildcard(r.File)
		if err != nil {
			return nil, err
		}

		r.Modifier = getString("modifier", tr, "")

		r.Text = getString("text", tr, "")
		if r.Text != "" {
			r.textRE, err = regexp.Compile(r.Text)
			if err != nil {
				return nil, err
			}
		}

		r.UserData = getString("userdata", tr, "")

		r.DeleteText = getBool("deletetext", tr, s.DeleteText)
		r.ExoFile = getString("exofile", tr, s.ExoFile)
		r.FileMove = getString("filemove", tr, s.FileMove)
		switch s.FileMove {
		case "off", "copy", "move":
			break
		default:
			r.FileMove = s.FileMove
		}
		r.LuaFile = getString("luafile", tr, s.LuaFile)
		r.Padding = getInt("padding", tr, s.Padding)

		s.Rule = append(s.Rule, r)
	}

	for _, tr := range getSubTreeArray("asas", config) {
		var a asas
		a.Exe = getString("exe", tr, "")

		flagDef := 1
		if f := tr.Get("format"); f == nil {
			flagDef = 3
		}
		a.Flags = getInt("flags", tr, flagDef)

		a.Filter = getString("filter", tr, "*.wav")

		a.Folder = getString("folder", tr, "%TEMPDIR%")
		a.Folder = dirReplacer.Replace(a.Folder)

		name := filepath.Base(a.Exe)
		formatDef := name[:len(name)-len(filepath.Ext(name))] + "_*.wav"
		a.Format = getString("format", tr, formatDef)

		s.Asas = append(s.Asas, a)
	}

	return &s, nil
}

var (
	shiftjis = japanese.ShiftJIS
	utf16le  = unicode.UTF16(unicode.LittleEndian, unicode.UseBOM)
	utf16be  = unicode.UTF16(unicode.BigEndian, unicode.UseBOM)
)

func (ss *setting) Find(path string) (*rule, string, error) {
	dir := filepath.Dir(path)
	base := filepath.Base(path)
	textRaw, err := ioutil.ReadFile(path[:len(path)-4] + ".txt")
	if err != nil {
		return nil, "", err
	}
	var u8, sjis, u16le, u16be *string

	for i := range ss.Rule {
		if verbose {
			log.Println("[INFO] ", i, "番目のルールを検証中...")
		}
		r := &ss.Rule[i]
		if dir != r.Dir {
			if verbose {
				log.Println("[INFO]   フォルダーのパスが一致しません")
				log.Println("[INFO]     want:", r.Dir)
				log.Println("[INFO]     got:", dir)
			}
			continue
		}
		if !r.fileRE.MatchString(base) {
			if verbose {
				log.Println("[INFO]   ファイル名がワイルドカードに一致しません")
				log.Println("[INFO]     filename:", base)
				log.Println("[INFO]     regex:", r.fileRE)
			}
			continue
		}
		if r.textRE != nil {
			switch r.Encoding {
			case "utf8":
				if u8 == nil {
					t := string(skipUTF8BOM(textRaw))
					u8 = &t
				}
				if !r.textRE.MatchString(*u8) {
					if verbose {
						log.Println("[INFO]     テキスト内容が正規表現にマッチしませんでした")
					}
					continue
				}
			case "sjis":
				if sjis == nil {
					b, err := shiftjis.NewDecoder().Bytes(textRaw)
					if err != nil {
						if verbose {
							log.Println("[INFO]     Shift_JIS から UTF-8 への文字コード変換に失敗しました")
							log.Println("[INFO]       ", err)
						}
						continue
					}
					t := string(b)
					sjis = &t
				}
				if !r.textRE.MatchString(*sjis) {
					if verbose {
						log.Println("[INFO]     テキスト内容が正規表現にマッチしませんでした")
					}
					continue
				}
			case "utf16le":
				if u16le == nil {
					b, err := utf16le.NewDecoder().Bytes(textRaw)
					if err != nil {
						if verbose {
							log.Println("[INFO]     UTF-16LE から UTF-8 への文字コード変換に失敗しました")
							log.Println("[INFO]       ", err)
						}
						continue
					}
					t := string(b)
					u16le = &t
				}
				if !r.textRE.MatchString(*u16le) {
					if verbose {
						log.Println("[INFO]     テキスト内容が正規表現にマッチしませんでした")
					}
					continue
				}
			case "utf16be":
				if u16be == nil {
					b, err := utf16be.NewDecoder().Bytes(textRaw)
					if err != nil {
						if verbose {
							log.Println("[INFO]     UTF-16BE から UTF-8 への文字コード変換に失敗しました")
							log.Println("[INFO]       ", err)
						}
						continue
					}
					t := string(b)
					u16be = &t
				}
				if !r.textRE.MatchString(*u16be) {
					if verbose {
						log.Println("[INFO]     テキスト内容が正規表現にマッチしませんでした")
					}
					continue
				}
			}
		}
		if verbose {
			log.Println("[INFO]   このルールに適合しました")
		}
		switch r.Encoding {
		case "utf8":
			if u8 == nil {
				t := string(skipUTF8BOM(textRaw))
				u8 = &t
			}
			return r, *u8, nil
		case "sjis":
			if sjis == nil {
				b, err := shiftjis.NewDecoder().Bytes(textRaw)
				if err != nil {
					return nil, "", fmt.Errorf("cannot convert encoding to shift_jis: %w", err)
				}
				t := string(b)
				sjis = &t
			}
			return r, *sjis, nil
		case "utf16le":
			if u16le == nil {
				b, err := utf16le.NewDecoder().Bytes(textRaw)
				if err != nil {
					return nil, "", fmt.Errorf("cannot convert encoding to utf-16le: %w", err)
				}
				t := string(b)
				u16le = &t
			}
			return r, *u16le, nil
		case "utf16be":
			if u16be == nil {
				b, err := utf16be.NewDecoder().Bytes(textRaw)
				if err != nil {
					return nil, "", fmt.Errorf("cannot convert encoding to utf-16be: %w", err)
				}
				t := string(b)
				u16be = &t
			}
			return r, *u16be, nil
		default:
			panic("unexcepted encoding value: " + r.Encoding)
		}
	}
	return nil, "", nil
}

func (ss *setting) Dirs() []string {
	dirs := map[string]struct{}{}
	for i := range ss.Rule {
		if ss.Rule[i].ExistsDir() {
			dirs[ss.Rule[i].Dir] = struct{}{}
		}
	}
	r := make([]string, 0, len(dirs))
	for k := range dirs {
		r = append(r, k)
	}
	sort.Strings(r)
	return r
}

func loadTOML(path string) (*toml.Tree, error) {
	b, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}
	return toml.LoadBytes(skipUTF8BOM(b))
}

func skipUTF8BOM(b []byte) []byte {
	if len(b) >= 3 && b[0] == 0xef && b[1] == 0xbb && b[2] == 0xbf {
		return b[3:]
	}
	return b
}
