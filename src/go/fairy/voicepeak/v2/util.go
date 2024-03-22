package voicepeak

import (
	"path/filepath"
)

func match(s string, patterns []string) bool {
	for _, ps := range patterns {
		if s == ps {
			return true
		}
	}
	return false
}

func changeExt(path, ext string) string {
	return path[:len(path)-len(filepath.Ext(path))] + ext
}
