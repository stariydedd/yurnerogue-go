//go:build !js

package sound

import "testing"

func setupTestSettingsStorage(t *testing.T) {
	t.Helper()
	config := t.TempDir()
	t.Setenv("APPDATA", config)
	t.Setenv("XDG_CONFIG_HOME", config)
}
