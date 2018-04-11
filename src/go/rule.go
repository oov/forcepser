package main

import (
	"os"
	"path/filepath"
	"regexp"

	toml "github.com/pelletier/go-toml"
)

type rule struct {
	Layer    int
	File     string
	Encoding string
	RE       *regexp.Regexp
}

type setting struct {
	Dir   string
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
	base := filepath.Base(path)
	for i := range ss.Rule {
		r := &ss.Rule[i]
		if r.RE.MatchString(base) {
			return r
		}
	}
	return nil
}
