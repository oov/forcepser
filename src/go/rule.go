package main

import (
	"bufio"
	"fmt"
	"io"
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

func exists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

type moveType string

func (mt moveType) Readable() string {
	switch mt {
	case "off":
		return "Off"
	case "copy":
		return "コピー"
	case "move":
		return "移動"
	}
	return fmt.Sprintf("無効な設定(%q)", string(mt))
}

type rule struct {
	Dir      string
	File     string
	FileRE   string
	Encoding string
	Layer    int
	Modifier string
	Text     string
	UserData string

	ExoFile    string
	LuaFile    string
	FileMove   moveType
	DestDir    string
	MoveDelay  float64
	DeleteText bool
	Padding    int

	fileRE      *regexp.Regexp
	textRE      *regexp.Regexp
	dirReplacer *strings.Replacer
}

func (r *rule) ExpandedDir() string {
	return r.dirReplacer.Replace(r.Dir)
}

func (r *rule) ExpandedDestDir() string {
	return r.dirReplacer.Replace(r.DestDir)
}

func (r *rule) ExistsDir() bool {
	return exists(r.ExpandedDir())
}

type setting struct {
	AcceptEmptyText bool
	BaseDir         string
	FileMove        moveType
	DeleteText      bool
	Delta           float64
	DestDir         string
	Freshness       float64
	MoveDelay       float64
	ExoFile         string
	LuaFile         string
	Padding         int
	Rule            []rule
	Asas            []asas

	Sort      string
	SortDelay float64

	FairyCall string

	projectDir  string
	dirReplacer *strings.Replacer
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

func newSetting(r io.Reader, tempDir string, projectDir string) (*setting, error) {
	config, err := loadTOML(r)
	if err != nil {
		return nil, fmt.Errorf("could not read setting file: %w", err)
	}
	var s setting
	s.projectDir = projectDir
	s.BaseDir = getString("basedir", config, "")
	s.dirReplacer = strings.NewReplacer(
		"%BASEDIR%", s.BaseDir,
		"%TEMPDIR%", tempDir,
		"%PROJECTDIR%", s.projectDir,
		"%PROFILE%", getSpecialFolderPath(CSIDL_PROFILE),
		"%DESKTOP%", getSpecialFolderPath(CSIDL_DESKTOP),
		"%MYDOC%", getSpecialFolderPath(CSIDL_PERSONAL),
	)

	s.Delta = getFloat64("delta", config, 15.0)
	s.Freshness = getFloat64("freshness", config, 5.0)
	s.MoveDelay = getFloat64("movedelay", config, 0)
	s.Padding = getInt("padding", config, 0)
	s.ExoFile = getString("exofile", config, "template.exo")
	s.LuaFile = getString("luafile", config, "genexo.lua")

	switch fm := getString("filemove", config, "off"); fm {
	case "off", "copy", "move":
		s.FileMove = moveType(fm)
	default:
		s.FileMove = moveType("off")
	}
	s.DestDir = getString("destdir", config, "%PROJECTDIR%")
	s.AcceptEmptyText = getBool("acceptemptytext", config, false)
	s.DeleteText = getBool("deletetext", config, false)

	switch ss := getString("sort", config, "moddate"); ss {
	case "moddate", "name":
		s.Sort = ss
	default:
		s.Sort = "moddate"
	}
	s.SortDelay = getFloat64("sortdelay", config, 0.1)

	s.FairyCall = getString("fairycall", config, "")

	for _, tr := range getSubTreeArray("rule", config) {
		var r rule
		r.dirReplacer = s.dirReplacer

		r.Dir = getString("dir", tr, "%TEMPDIR%")

		r.Encoding = getString("encoding", tr, "sjis")

		r.Layer = getInt("layer", tr, 1)

		r.File = getString("file", tr, "")
		r.FileRE = getString("filere", tr, "")
		if r.File != "" && r.FileRE != "" {
			return nil, fmt.Errorf("file and fileRE cannot be used at the same time")
		}
		if r.FileRE != "" {
			r.fileRE, err = regexp.Compile(r.FileRE)
		} else {
			if r.File == "" {
				r.File = "*.wav"
			}
			r.fileRE, err = makeWildcard(r.File)
		}
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
		switch fm := getString("filemove", tr, string(s.FileMove)); fm {
		case "off", "copy", "move":
			r.FileMove = moveType(fm)
		default:
			r.FileMove = s.FileMove
		}
		r.DestDir = getString("destdir", tr, s.DestDir)
		r.MoveDelay = getFloat64("movedelay", tr, s.MoveDelay)
		r.LuaFile = getString("luafile", tr, s.LuaFile)
		r.Padding = getInt("padding", tr, s.Padding)

		s.Rule = append(s.Rule, r)
	}

	for _, tr := range getSubTreeArray("asas", config) {
		var a asas
		a.dirReplacer = s.dirReplacer
		a.Exe = getString("exe", tr, "")

		flagDef := 1
		if f := tr.Get("format"); f == nil {
			flagDef = 3
		}
		a.Flags = getInt("flags", tr, flagDef)

		a.Filter = getString("filter", tr, "*.wav")

		a.Folder = getString("folder", tr, "%TEMPDIR%")

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
	dirFI, err := getFileInfo(dir)
	if err != nil {
		return nil, "", fmt.Errorf("failed to get directory info: %w", err)
	}

	base := filepath.Base(path)
	textRaw, err := os.ReadFile(path[:len(path)-4] + ".txt")
	if err != nil {
		return nil, "", err
	}
	var u8, sjis, u16le, u16be *string

	for i := range ss.Rule {
		if verbose {
			log.Println(suppress.Renderln(i, "番目のルールを検証中..."))
		}
		r := &ss.Rule[i]
		ruleDir := r.ExpandedDir()
		ruledirFI, err := getFileInfo(ruleDir)
		if err != nil {
			if verbose {
				log.Println(suppress.Renderln("  フォルダーの情報取得に失敗しました"))
				log.Println(suppress.Renderln("    dir:", ruleDir))
				log.Println(suppress.Renderln("    error:", err))
			}
			continue
		}
		if !isSameFileInfo(dirFI, ruledirFI) {
			if verbose {
				log.Println(suppress.Renderln("  フォルダーが一致しません"))
				log.Println(suppress.Renderln("    want:", r.ExpandedDir()))
				log.Println(suppress.Renderln("    got:", dir))
			}
			continue
		}
		if !r.fileRE.MatchString(base) {
			if verbose {
				log.Println(suppress.Renderln("  ファイル名がワイルドカードに一致しません"))
				log.Println(suppress.Renderln("    filename:", base))
				log.Println(suppress.Renderln("    regex:", r.fileRE))
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
						log.Println(suppress.Renderln("    テキスト内容が正規表現にマッチしませんでした"))
					}
					continue
				}
			case "sjis":
				if sjis == nil {
					b, err := shiftjis.NewDecoder().Bytes(textRaw)
					if err != nil {
						if verbose {
							log.Println(suppress.Renderln("    Shift_JIS から UTF-8 への文字コード変換に失敗しました"))
							log.Println(suppress.Renderln("      ", err))
						}
						continue
					}
					t := string(b)
					sjis = &t
				}
				if !r.textRE.MatchString(*sjis) {
					if verbose {
						log.Println(suppress.Renderln("    テキスト内容が正規表現にマッチしませんでした"))
					}
					continue
				}
			case "utf16le":
				if u16le == nil {
					b, err := utf16le.NewDecoder().Bytes(textRaw)
					if err != nil {
						if verbose {
							log.Println(suppress.Renderln("    UTF-16LE から UTF-8 への文字コード変換に失敗しました"))
							log.Println(suppress.Renderln("      ", err))
						}
						continue
					}
					t := string(b)
					u16le = &t
				}
				if !r.textRE.MatchString(*u16le) {
					if verbose {
						log.Println(suppress.Renderln("    テキスト内容が正規表現にマッチしませんでした"))
					}
					continue
				}
			case "utf16be":
				if u16be == nil {
					b, err := utf16be.NewDecoder().Bytes(textRaw)
					if err != nil {
						if verbose {
							log.Println(suppress.Renderln("    UTF-16BE から UTF-8 への文字コード変換に失敗しました"))
							log.Println(suppress.Renderln("      ", err))
						}
						continue
					}
					t := string(b)
					u16be = &t
				}
				if !r.textRE.MatchString(*u16be) {
					if verbose {
						log.Println(suppress.Renderln("    テキスト内容が正規表現にマッチしませんでした"))
					}
					continue
				}
			}
		}
		if verbose {
			log.Println(suppress.Renderln("  このルールに適合しました"))
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
			dirs[ss.Rule[i].ExpandedDir()] = struct{}{}
		}
	}
	r := make([]string, 0, len(dirs))
	for k := range dirs {
		r = append(r, k)
	}
	sort.Strings(r)
	return r
}

func loadTOML(r io.Reader) (*toml.Tree, error) {
	rr := bufio.NewReader(r)
	b, err := rr.Peek(3)
	if err == nil && isUTF8BOM(b) {
		if _, err = rr.Discard(3); err != nil {
			return nil, err
		}
	}
	return toml.LoadReader(rr)
}

func isUTF8BOM(b []byte) bool {
	return len(b) >= 3 && b[0] == 0xef && b[1] == 0xbb && b[2] == 0xbf
}

func skipUTF8BOM(b []byte) []byte {
	if isUTF8BOM(b) {
		return b[3:]
	}
	return b
}
