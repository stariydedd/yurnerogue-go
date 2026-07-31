package domain

import "strconv"

// itoa — короткая обёртка для сборки сообщений в HUD.
func itoa(v int) string { return strconv.Itoa(v) }

func abs(v int) int {
	if v < 0 {
		return -v
	}
	return v
}

func clamp(v, lo, hi int) int {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}
