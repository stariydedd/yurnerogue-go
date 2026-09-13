package sound

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"math"
	"testing"

	"github.com/hajimehoshi/ebiten/v2/audio/wav"
)

func TestEmbeddedCombatSoundsKeepRecordingTimingAndTimbre(t *testing.T) {
	clips, err := loadAttackSamples()
	if err != nil {
		t.Fatal(err)
	}
	misses, err := loadMissSamples()
	if err != nil {
		t.Fatal(err)
	}
	for _, group := range []struct {
		family                   string
		clips                    [][]byte
		minDuration, maxDuration float64
	}{
		{"attack", clips[:], 0.3, 0.6},
		{"miss", misses[:], 0.1, 0.25},
	} {
		for i, pcm := range group.clips {
			checkPCM(t, pcm)
			seconds := float64(len(pcm)) / float64(SampleRate*4)
			if seconds < group.minDuration || seconds > group.maxDuration {
				t.Fatal("recording truncated or wrong playback rate")
			}
			for j := 0; j < i; j++ {
				if bytes.Equal(pcm, group.clips[j]) {
					t.Fatal("duplicate attack recording")
				}
			}
			encoded, err := combatRecordings.ReadFile(fmt.Sprintf("clips/%s-%d.wav", group.family, i+1))
			if err != nil {
				t.Fatal(err)
			}
			stream, err := wav.DecodeWithoutResampling(bytes.NewReader(encoded))
			if err != nil {
				t.Fatal(err)
			}
			if stream.SampleRate() != SampleRate {
				t.Fatal("unexpected sample rate")
			}
			source, err := io.ReadAll(stream)
			if err != nil {
				t.Fatal(err)
			}
			if len(source) != len(pcm) {
				t.Fatal("playback timing changed")
			}
			// Outside the tiny boundary fades, every sample is just the original
			// recording at a lower level: no synthesis, equalization or pitch shift.
			margin := (SampleRate / 100) * 4
			for offset := margin; offset < len(source)-margin; offset += 2 {
				original := int16(binary.LittleEndian.Uint16(source[offset:]))
				got := int16(binary.LittleEndian.Uint16(pcm[offset:]))
				want := int16(float64(original) * recordedCombatGain)
				if got != want {
					t.Fatal("recording was filtered or replaced")
				}
			}
			if !bytes.Equal(pcm[:4], make([]byte, 4)) || !bytes.Equal(pcm[len(pcm)-4:], make([]byte, 4)) {
				t.Fatal("recording has abrupt boundary")
			}
		}
	}
	if effect(Hit) != nil || effect(Swing) != nil {
		t.Fatal("synthesized combat sound still exists")
	}
	if 0.8*musicGain+maxVoices*effectsGain*math.Max(0.8, recordedCombatGain) > 1 {
		t.Fatal("recordings can overload mixer")
	}
}

func TestCombatSelectionUsesEveryVariantWithoutRepeats(t *testing.T) {
	for _, count := range []int{attackVariants, missVariants} {
		var bag variantBag
		previous := -1
		for round := 0; round < 1000; round++ {
			seen := map[int]bool{}
			for i := 0; i < count; i++ {
				v := bag.next(count)
				if v == previous || v < 0 || v >= count || seen[v] {
					t.Fatal("attack recording repeats or is out of range")
				}
				seen[v], previous = true, v
			}
		}
	}
}

func TestMutedOrThrottledAttacksDoNotConsumeRecording(t *testing.T) {
	for _, e := range []*Engine{
		{bank: &bank{}},
		{settings: Defaults()},
		{bank: &bank{}, settings: Defaults(), tick: 4, last: [cueCount]int{Hit: 3, Swing: 3}},
	} {
		e.Play(Hit)
		e.Play(Swing)
		if e.attackBag != (variantBag{}) || e.missBag != (variantBag{}) {
			t.Fatal("skipped attack consumed recording")
		}
	}
}
