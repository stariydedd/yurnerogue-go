package sound

import (
	"bytes"
	"encoding/binary"
	"testing"
)

func peak(pcm []byte) float64 {
	best := 0
	for i := 0; i+1 < len(pcm); i += 2 {
		v := int(int16(binary.LittleEndian.Uint16(pcm[i:])))
		best = max(best, v, -v)
	}
	return float64(best) / 32768
}

func TestVictoryIsItsOwnFanfare(t *testing.T) {
	victory, start := effect(Victory), effect(Start)
	if bytes.Equal(victory, start) {
		t.Fatal("victory still sounds like the start of a run")
	}
	if seconds := float64(len(victory)/4) / SampleRate; seconds < 3.5 {
		t.Fatalf("fanfare lasts %.1f s, want a longer finale", seconds)
	}
	if p, s := peak(victory), peak(start); p > .98 || p < s*.8 {
		t.Fatalf("fanfare peak %.2f against start %.2f: clipped or too quiet", p, s)
	}
}

type fakeTrack struct {
	volume  float64
	playing bool
}

func (f *fakeTrack) Play()               { f.playing = true }
func (f *fakeTrack) Pause()              { f.playing = false }
func (f *fakeTrack) IsPlaying() bool     { return f.playing }
func (f *fakeTrack) SetVolume(v float64) { f.volume = v }

func TestVictorySilencesTheForestMusicAndItComesBack(t *testing.T) {
	setupTestSettingsStorage(t)
	menu, world := &fakeTrack{}, &fakeTrack{}
	e := &Engine{bank: &bank{}, tracks: [2]musicTrack{menu, world}, settings: Settings{Music: 100, Effects: 100}}
	for i := 0; i < 300; i++ {
		e.Update(true, true)
	}
	if world.volume == 0 {
		t.Fatal("forest music never started")
	}
	e.SilenceMusic(true)
	for i := 0; i < 20; i++ {
		e.Update(true, true)
	}
	if world.volume != 0 || menu.volume != 0 {
		t.Fatalf("music still at %.3f a third of a second into the fanfare", world.volume)
	}
	e.SilenceMusic(false)
	e.Update(false, true)
	if menu.volume == 0 || menu.volume > .01 {
		t.Fatalf("menu music should fade back in gently, got %.3f", menu.volume)
	}
}
