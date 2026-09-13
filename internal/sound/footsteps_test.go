package sound

import (
	"bytes"
	"encoding/binary"
	"math"
	"testing"
)

func TestStepBagsNeverRepeatAndUseEveryVariant(t *testing.T) {
	var bags [2]variantBag
	last := [2]int{-1, -1}
	// Interleave surfaces, including returning after a long stretch elsewhere.
	for round := 0; round < 1000; round++ {
		for _, surface := range []int{0, 1, 0, 0, 1} {
			seen := map[int]bool{}
			for i := 0; i < stepVariants; i++ {
				v := bags[surface].next(stepVariants)
				if v < 0 || v >= stepVariants || v == last[surface] || seen[v] {
					t.Fatalf("repeated/invalid variant on surface %d: %d", surface, v)
				}
				seen[v], last[surface] = true, v
			}
		}
	}
}

func TestStepVariantsAreDistinctQuietAndShort(t *testing.T) {
	var all [][]byte
	for _, c := range []Cue{StepGrass, StepTrail} {
		for variant := 0; variant < stepVariants; variant++ {
			data := footstep(c, variant)
			checkPCM(t, data)
			if len(data) > SampleRate*4/4 || !bytes.Equal(data, footstep(c, variant)) {
				t.Fatal("step is too long or nondeterministic")
			}
			if !bytes.Equal(data[:4], make([]byte, 4)) || !bytes.Equal(data[len(data)-4:], make([]byte, 4)) {
				t.Fatal("abrupt step endpoint")
			}
			for i := 0; i < len(data); i += 2 {
				if math.Abs(float64(int16(binary.LittleEndian.Uint16(data[i:])))) > 0.22*32767 {
					t.Fatal("footstep is as loud as combat")
				}
			}
			for _, other := range all {
				if bytes.Equal(data, other) {
					t.Fatal("duplicate footstep sample")
				}
			}
			all = append(all, data)
		}
	}
}

func TestSkippedStepsDoNotAdvanceVariantBags(t *testing.T) {
	for _, e := range []*Engine{
		{bank: &bank{}},        // Muted.
		{settings: Defaults()}, // Still generating.
		{bank: &bank{}, settings: Defaults(), stepPlayed: true, tick: 3}, // Cadence gate.
	} {
		for _, cue := range []Cue{StepGrass, StepTrail} {
			e.Play(cue)
		}
		if e.stepBags != ([2]variantBag{}) {
			t.Fatal("inaudible step consumed a variant")
		}
	}
}
