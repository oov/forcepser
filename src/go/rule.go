package main

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"

	toml "github.com/pelletier/go-toml"
	"github.com/pkg/errors"
	"golang.org/x/text/encoding/japanese"
)

type rule struct {
	Dir      string
	File     string
	Text     string
	Encoding string
	Layer    int
	Modifier string

	fileRE *regexp.Regexp
	textRE *regexp.Regexp
}

type setting struct {
	BaseDir   string
	Delta     float64
	Freshness float64
	Rule      []rule
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

func newSetting(path string) (*setting, error) {
	f, err := openTextFile(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var s setting
	config, err := toml.LoadReader(f)
	if err != nil {
		return nil, err
	}
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
		r.Dir = strings.NewReplacer("%BASEDIR%", s.BaseDir).Replace(r.Dir)
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
	return &s, nil
}

func (ss *setting) Find(path string) (*rule, string, error) {
	dir := filepath.Dir(path)
	base := filepath.Base(path)
	textRaw, err := readFile(path[:len(path)-4] + ".txt")
	if err != nil {
		return nil, "", err
	}
	var u8, sjis *string

	for i := range ss.Rule {
		r := &ss.Rule[i]
		if dir != r.Dir {
			continue
		}
		if !r.fileRE.MatchString(base) {
			continue
		}
		if r.textRE != nil {
			switch r.Encoding {
			case "utf8":
				if u8 == nil {
					t := string(textRaw)
					u8 = &t
				}
				if !r.textRE.MatchString(*u8) {
					continue
				}
			case "sjis":
				if sjis == nil {
					b, err := japanese.ShiftJIS.NewDecoder().Bytes(textRaw)
					if err != nil {
						continue // this file is not written in Shift_JIS.
					}
					t := string(b)
					sjis = &t
				}
				if !r.textRE.MatchString(*sjis) {
					continue
				}
			}
		}
		switch r.Encoding {
		case "utf8":
			if u8 == nil {
				t := string(textRaw)
				u8 = &t
			}
			return r, *u8, nil
		case "sjis":
			if sjis == nil {
				b, err := japanese.ShiftJIS.NewDecoder().Bytes(textRaw)
				if err != nil {
					return nil, "", errors.Wrap(err, "cannot convert encoding to shift_jis")
				}
				t := string(b)
				sjis = &t
			}
			return r, *sjis, nil
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

func readFile(path string) ([]byte, error) {
	f, err := openTextFile(path)
	if err != nil {
		return nil, errors.Wrap(err, "failed to openTextFile")
	}
	defer f.Close()

	b, err := ioutil.ReadAll(f)
	if err != nil {
		return nil, errors.Wrap(err, "ioutil.ReadAll failed")
	}
	return b, nil
}

func openTextFile(path string) (*os.File, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, errors.Wrap(err, "os.Open failed")
	}

	// skip BOM
	var bom [3]byte
	_, err = f.ReadAt(bom[:], 0)
	if err != nil {
		f.Close()
		return nil, errors.Wrap(err, "cannot read first 3bytes")
	}
	if bom[0] == 0xef && bom[1] == 0xbb && bom[2] == 0xbf {
		_, err = f.Seek(3, os.SEEK_SET)
		if err != nil {
			f.Close()
			return nil, errors.Wrap(err, "failed to seek")
		}
	}
	return f, nil
}
