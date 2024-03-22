package voisonatalk

func match(s string, patterns []string) bool {
	for _, ps := range patterns {
		if s == ps {
			return true
		}
	}
	return false
}
