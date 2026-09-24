package sound

import (
	"bytes"
	"encoding/binary"
	"math"
	"testing"
)

func checkPCM(t *testing.T, data []byte) {
	t.Helper()
	if len(data) == 0 || len(data)%4 != 0 {
		t.Fatal("invalid stereo PCM length")
	}
	energy := 0.0
	for i := 0; i < len(data); i += 2 {
		v := float64(int16(binary.LittleEndian.Uint16(data[i:])))
		if math.Abs(v) > 0.81*32767 {
			t.Fatal("missing audio headroom")
		}
		energy += v * v
	}
	if energy == 0 {
		t.Fatal("silent audio")
	}
}

func TestEffectsAreDeterministicAndHaveSoftEndpoints(t *testing.T) {
	for c := Cue(0); c < cueCount; c++ {
		if Recorded(c) {
			continue
		} // Covered by embedded recording tests.
		data := effect(c)
		checkPCM(t, data)
		if !bytes.Equal(data, effect(c)) {
			t.Fatal("nondeterministic effect")
		}
		if !bytes.Equal(data[:4], make([]byte, 4)) || !bytes.Equal(data[len(data)-4:], make([]byte, 4)) {
			t.Fatalf("cue %d has an abrupt endpoint", c)
		}
	}
}

func TestMusicLoopsHaveMatchingFormatAndNoLargeSeam(t *testing.T) {
	a, b := music(false), music(true)
	if len(a) != 32*SampleRate*4 || len(a) != len(b) || bytes.Equal(a, b) {
		t.Fatal("music arrangements missing or mismatched")
	}
	for _, data := range [][]byte{a, b} {
		checkPCM(t, data)
		for channel := 0; channel < 2; channel++ {
			first := int16(binary.LittleEndian.Uint16(data[channel*2:]))
			last := int16(binary.LittleEndian.Uint16(data[len(data)-4+channel*2:]))
			if math.Abs(float64(first)-float64(last)) > 1800 {
				t.Fatal("audible discontinuity at loop seam")
			}
		}
	}
}

func TestVolumeSettings(t *testing.T) {
	if !Defaults().Valid() || (Settings{-1, 50}).Valid() || (Settings{50, 101}).Valid() {
		t.Fatal("invalid volume bounds")
	}
	v := 0
	for _, want := range []int{25, 50, 75, 100, 0} {
		v = NextVolume(v)
		if v != want {
			t.Fatal("volume cycle does not include mute")
		}
	}
	if musicGain*0.8+maxVoices*effectsGain*0.8 > 1 {
		t.Fatal("worst-case mixer can clip")
	}
}
