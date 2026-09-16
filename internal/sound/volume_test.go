package sound

import "testing"

func TestSetVolumeClampsAndPersistsBothChannels(t *testing.T) {
	setupTestSettingsStorage(t)
	e := &Engine{settings: Defaults()}
	e.SetVolume(true, 37)
	e.SetVolume(false, 63)
	if e.Settings() != (Settings{Music: 37, Effects: 63}) || loadSettings() != e.Settings() {
		t.Fatal("slider values were not preserved")
	}
	e.SetVolume(true, -5)
	e.SetVolume(false, 105)
	if e.Settings() != (Settings{Music: 0, Effects: 100}) || loadSettings() != e.Settings() {
		t.Fatal("volume bounds not preserved")
	}
	var disabled *Engine
	disabled.SetVolume(true, 50)
}
