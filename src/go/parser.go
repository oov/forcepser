package main

import (
	"fmt"
	"strconv"

	"github.com/pelletier/go-toml"
)

func getString(key string, t *toml.Tree, def string) string {
	v := t.Get(key)
	if v == nil {
		return def
	}
	return toString(v)
}

func getBool(key string, t *toml.Tree, def bool) bool {
	v := t.Get(key)
	if v == nil {
		return def
	}
	return toBool(v)
}

func getFloat64(key string, t *toml.Tree, def float64) float64 {
	v := t.Get(key)
	if v == nil {
		return def
	}
	f, err := toFloat64(v)
	if err != nil {
		return def
	}
	return f
}

func getInt(key string, t *toml.Tree, def int) int {
	v := t.Get(key)
	if v == nil {
		return def
	}
	i, err := toInt(v)
	if err != nil {
		return def
	}
	return i
}

func getSubTreeArray(key string, t *toml.Tree) []*toml.Tree {
	r, ok := t.Get(key).([]*toml.Tree)
	if !ok {
		return []*toml.Tree{}
	}
	return r
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
	return 0, fmt.Errorf("unexpected value type: %T", v)
}

func toInt(v interface{}) (int, error) {
	f, err := toFloat64(v)
	if err != nil {
		return 0, err
	}
	return int(f), nil
}

func toString(v interface{}) string {
	s, ok := v.(string)
	if ok {
		return s
	}
	return fmt.Sprint(v)
}

func toBool(v interface{}) bool {
	b, ok := v.(bool)
	if ok {
		return b
	}
	s, ok := v.(string)
	if ok {
		return s == "true" || s == "True" || s == "TRUE"
	}
	f, err := toFloat64(v)
	if err != nil {
		return false
	}
	return f != 0
}
