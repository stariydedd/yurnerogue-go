//go:build !js

package sound

import (
	"encoding/json"
	"os"
	"path/filepath"
)

func settingsPath() string {
	dir, err := os.UserConfigDir()
	if err != nil {
		return ""
	}
	return filepath.Join(dir, "yurnerogue", "audio.json")
}
func loadSettings() Settings {
	s := Defaults()
	data, err := os.ReadFile(settingsPath())
	if err != nil || json.Unmarshal(data, &s) != nil || !s.Valid() {
		return Defaults()
	}
	return s
}
func saveSettings(s Settings) {
	path := settingsPath()
	if path == "" {
		return
	}
	if os.MkdirAll(filepath.Dir(path), 0700) != nil {
		return
	}
	data, _ := json.Marshal(s)
	_ = os.WriteFile(path, data, 0600)
}
