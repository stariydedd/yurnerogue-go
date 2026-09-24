package sound

import (
	"encoding/binary"
	"math"
	"testing"
)

func TestFloat32PCMKeepsLevelsAndSigns(t *testing.T) {
	samples := []int16{0, 32767, -32768, -1, 16384}
	pcm := make([]byte, len(samples)*2)
	for i, s := range samples {
		binary.LittleEndian.PutUint16(pcm[i*2:], uint16(s))
	}
	out := float32PCM(pcm)
	if len(out) != len(pcm)*2 {
		t.Fatalf("got %d bytes, want %d", len(out), len(pcm)*2)
	}
	for i, s := range samples {
		got := math.Float32frombits(binary.LittleEndian.Uint32(out[i*4:]))
		if want := float32(s) / 32768; got != want {
			t.Fatalf("sample %d: got %v, want %v", i, got, want)
		}
	}
}

func TestBankEffectsAreFloat32(t *testing.T) {
	b := bank{}
	b.cues[Hit] = make([]byte, 8)
	b.attacks[0] = make([]byte, 6)
	b.steps[0][0] = make([]byte, 4)
	b.misses[0] = make([]byte, 2)
	b.menu = make([]byte, 10)
	b.toFloat32()
	if len(b.cues[Hit]) != 16 || len(b.attacks[0]) != 12 || len(b.steps[0][0]) != 8 || len(b.misses[0]) != 4 {
		t.Fatal("an effect clip was not converted to float32")
	}
	if len(b.menu) != 10 {
		t.Fatal("music must stay 16-bit")
	}
}
