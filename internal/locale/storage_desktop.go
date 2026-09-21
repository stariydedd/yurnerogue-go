//go:build !js

package locale

import (
	"os"
	"path/filepath"
)

func languagePath() string {
	dir, err := os.UserConfigDir()
	if err != nil {
		return ""
	}
	return filepath.Join(dir, "yurnerogue", "language")
}

func Load() Language {
	data, _ := os.ReadFile(languagePath())
	return Normalize(Language(data))
}

func Save(language Language) {
	path := languagePath()
	if path == "" || os.MkdirAll(filepath.Dir(path), 0700) != nil {
		return
	}
	_ = os.WriteFile(path, []byte(Normalize(language)), 0600)
}
