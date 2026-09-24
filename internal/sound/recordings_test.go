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
		gain                     float64
	}{
		{"attack", clips[:], 0.3, 0.6, attackGain},
		{"miss", misses[:], 0.1, 0.25, missGain},
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
				want := int16(float64(original) * group.gain)
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
	if 0.8*musicGain+maxVoices*effectsGain*math.Max(0.8, max(attackGain, missGain)) > 1 {
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

func TestAbilityRecordingsLoadWithTheirLength(t *testing.T) {
	for c, want := range map[Cue]float64{Critical: 1.39, Parry: 0.56} {
		clip := abilityRecordings[c]
		pcm, err := loadRecording(clip.name, clip.gain)
		if err != nil {
			t.Fatal(err)
		}
		if got := float64(len(pcm)/4) / SampleRate; math.Abs(got-want) > 0.05 {
			t.Fatalf("cue %d lasts %.2fs, want about %.2fs", c, got, want)
		}
		if !Recorded(c) || effect(c) != nil {
			t.Fatalf("cue %d must be played from its recording, not synthesized", c)
		}
	}
}

// Loudness is raised in playback, so the stored clip must never be clipped or
// squashed, and the extra gain must stay inside the mixer headroom budget.
func TestAbilityRecordingsAreCleanAndInsideHeadroom(t *testing.T) {
	for c, clip := range abilityRecordings {
		if clip.gain > 0.8 || clip.gain < recordedCombatGain {
			t.Fatalf("cue %d gain %.2f outside the attack-to-headroom range", c, clip.gain)
		}
		raw, err := loadRecording(clip.name, 1)
		if err != nil {
			t.Fatal(err)
		}
		peak, full := 0, 0
		for i := 0; i < len(raw); i += 2 {
			v := int(int16(binary.LittleEndian.Uint16(raw[i:])))
			peak = max(peak, max(v, -v))
			if v >= 32767 || v <= -32767 {
				full++
			}
		}
		if full > 0 || peak > 32200 {
			t.Fatalf("cue %d file is clipped or has no headroom: peak %d, %d full-scale samples", c, peak, full)
		}
	}
}
