package main

import (
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"

	toml "github.com/pelletier/go-toml"
	"github.com/pkg/errors"
	"golang.org/x/text/encoding/japanese"
	"golang.org/x/text/encoding/unicode"
)

type rule struct {
	Dir      string `default:"%TEMPDIR%"`
	File     string `default:"*.wav"`
	Text     string
	Encoding string `default:"sjis"`
	Layer    int    `default:"1"`
	Modifier string
	UserData string

	fileRE *regexp.Regexp
	textRE *regexp.Regexp
}

type setting struct {
	BaseDir    string
	FileMove   string
	DeleteText bool
	Delta      float64
	Freshness  float64
	Rule       []rule
	Asas       []asas
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

func decodeTOML(r io.Reader, v interface{}) (err error) {
	defer func() {
		if rcv := recover(); rcv != nil {
			err = fmt.Errorf("failed to decode TOML: %v", rcv)
		}
	}()
	return toml.NewDecoder(r).Decode(v)
}

func toFloat64(v interface{}) (float64, error) {
	switch vv := v.(type) {
	case float64:
		return vv, nil
	case float32:
		return float64(vv), nil
	case int64:
		return float64(vv), nil
	case uint64:
		return float64(vv), nil
	case int32:
		return float64(vv), nil
	case uint32:
		return float64(vv), nil
	case int16:
		return float64(vv), nil
	case uint16:
		return float64(vv), nil
	case int8:
		return float64(vv), nil
	case uint8:
		return float64(vv), nil
	case int:
		return float64(vv), nil
	case uint:
		return float64(vv), nil
	case string:
		return strconv.ParseFloat(vv, 64)
	}
	return 0, errors.Errorf("unexpected value type: %T", v)
}

func tomlError(err error, tree *toml.Tree, key string) error {
	if err == nil {
		return nil
	}
	pos := tree.GetPosition(key)
	if pos.Invalid() {
		return err
	}
	return errors.Wrapf(err, "%s(%v行目)", key, pos.Line)
}

func newSetting(path string, tempDir string) (*setting, error) {
	config, err := loadTOML(path)
	if err != nil {
		return nil, errors.Wrap(err, "could not read setting file")
	}
	var s setting
	s.BaseDir, _ = config.GetDefault("basedir", "").(string)
	s.Delta, err = toFloat64(config.GetDefault("delta", 15.0))
	if err != nil {
		return nil, tomlError(err, config, "delta")
	}
	s.Freshness, err = toFloat64(config.GetDefault("freshness", 5.0))
	if err != nil {
		return nil, tomlError(err, config, "freshness")
	}
	var rules struct {
		Rule []rule
	}
	err = config.Unmarshal(&rules)
	if err != nil {
		return nil, tomlError(err, config, "rule")
	}
	s.Rule = rules.Rule
	for i := range s.Rule {
		r := &s.Rule[i]
		r.Dir = strings.NewReplacer("%BASEDIR%", s.BaseDir, "%TEMPDIR%", tempDir).Replace(r.Dir)
		r.fileRE, err = makeWildcard(r.File)
		if err != nil {
			return nil, err
		}
		if r.Text != "" {
			r.textRE, err = regexp.Compile(r.Text)
			if err != nil {
				return nil, err
			}
		}
	}

	var asas struct {
		Asas []asas
	}
	err = config.Unmarshal(&asas)
	if err != nil {
		return nil, tomlError(err, config, "asas")
	}
	s.Asas = asas.Asas
	for i := range s.Asas {
		a := &s.Asas[i]
		a.Folder = strings.NewReplacer("%BASEDIR%", s.BaseDir, "%TEMPDIR%", tempDir).Replace(a.Folder)
		if a.Flags == -1 {
			if a.Format == "" {
				a.Flags = 3
			} else {
				a.Flags = 1
			}
		}
		if a.Format == "" {
			name := filepath.Base(a.Exe)
			a.Format = name[:len(name)-len(filepath.Ext(name))] + "_*.wav"
		}
	}

	var fileMoveDefault string
	var deleteTextDefault bool
	if len(s.Asas) == 0 {
		fileMoveDefault = "off"
		deleteTextDefault = false
	} else {
		fileMoveDefault = "move"
		deleteTextDefault = true
	}
	s.FileMove, _ = config.GetDefault("filemove", fileMoveDefault).(string)
	switch s.FileMove {
	case "off", "copy", "move":
		break
	default:
		return nil, fmt.Errorf("unexpected filemove value: %q", s.FileMove)
	}
	s.DeleteText, _ = config.GetDefault("deletetext", deleteTextDefault).(bool)

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
					return nil, "", errors.Wrap(err, "cannot convert encoding to shift_jis")
				}
				t := string(b)
				sjis = &t
			}
			return r, *sjis, nil
		case "utf16le":
			if u16le == nil {
				b, err := utf16le.NewDecoder().Bytes(textRaw)
				if err != nil {
					return nil, "", errors.Wrap(err, "cannot convert encoding to utf-16le")
				}
				t := string(b)
				u16le = &t
			}
			return r, *u16le, nil
		case "utf16be":
			if u16be == nil {
				b, err := utf16be.NewDecoder().Bytes(textRaw)
				if err != nil {
					return nil, "", errors.Wrap(err, "cannot convert encoding to utf-16be")
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
		dirs[ss.Rule[i].Dir] = struct{}{}
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
		return nil, errors.Wrap(err, "failed to read file")
	}
	return toml.LoadBytes(skipUTF8BOM(b))
}

func skipUTF8BOM(b []byte) []byte {
	if len(b) >= 3 && b[0] == 0xef && b[1] == 0xbb && b[2] == 0xbf {
		return b[3:]
	}
	return b
}
