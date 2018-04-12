package main

import (
	"os"
	"path/filepath"
	"regexp"
	"sort"

	toml "github.com/pelletier/go-toml"
)

type rule struct {
	Dir      string
	File     string
	Encoding string
	Layer    int
	RE       *regexp.Regexp
}

type setting struct {
	Delta float64
	Rule  []rule
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

func newSetting(path string) (*setting, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	// skip BOM
	var bom [3]byte
	_, err = f.ReadAt(bom[:], 0)
	if err != nil {
		return nil, err
	}
	if bom[0] == 0xef && bom[1] == 0xbb && bom[2] == 0xbf {
		_, err = f.Seek(3, os.SEEK_SET)
		if err != nil {
			return nil, err
		}
	}

	var s setting
	err = toml.NewDecoder(f).Decode(&s)
	if err != nil {
		return nil, err
	}
	for i := range s.Rule {
		s.Rule[i].RE, err = makeWildcard(s.Rule[i].File)
		if err != nil {
			return nil, err
		}
	}
	return &s, nil
}

func (ss *setting) Find(path string) *rule {
	dir := filepath.Dir(path)
	base := filepath.Base(path)
	for i := range ss.Rule {
		r := &ss.Rule[i]
		if dir == r.Dir && r.RE.MatchString(base) {
			return r
		}
	}
	return nil
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
