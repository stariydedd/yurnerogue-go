package sound

import (
	"math"
	"testing"
)

// referenceTone is tone written out directly, one math.Sin and math.Exp per
// partial per sample: the formula tone must keep matching.
func referenceTone(dst []float64, at, duration, hz, gain, attack float64, bell bool) {
	for i := 0; i < int(duration*SampleRate); i++ {
		t := float64(i) / SampleRate
		env := math.Min(1, t/attack) * math.Exp(-3*t/duration)
		env *= math.Min(1, (duration-t)/0.08)
		x := math.Sin(2 * math.Pi * hz * t)
		if bell {
			x += 0.28 * math.Sin(2*math.Pi*hz*2.01*t) * math.Exp(-3*t)
			x += 0.09 * math.Sin(2*math.Pi*hz*3.98*t) * math.Exp(-5*t)
		} else {
			x += 0.22*math.Sin(2*math.Pi*hz*2*t) + 0.08*math.Sin(2*math.Pi*hz*3*t)
		}
		dst[(int(at*SampleRate)+i)%len(dst)] += x * env * gain
	}
}

func TestToneMatchesItsFormula(t *testing.T) {
	for _, n := range []struct {
		at, duration, hz, gain, attack float64
		bell                           bool
	}{
		{0, 12, 73.4162, 0.10, 2, false},  // a long music drone
		{30.5, 4, 440, 0.11, 0.025, true}, // a harp note wrapping round the loop
		{0.24, 0.42, 987.77, 0.16, 0.006, true},
		{1.45, 2.8, 1760, 0.06, 0.03, false}, // the fanfare's top note
	} {
		got, want := make([]float64, 32*SampleRate), make([]float64, 32*SampleRate)
		tone(got, n.at, n.duration, n.hz, n.gain, n.attack, n.bell)
		referenceTone(want, n.at, n.duration, n.hz, n.gain, n.attack, n.bell)
		worst := 0.0
		for i := range got {
			worst = max(worst, math.Abs(got[i]-want[i]))
		}
		// 1e-9 of full scale is far below a 16-bit sample step (3e-5).
		if worst > 1e-9 {
			t.Errorf("%.2f Hz from %.2f s: off the formula by %g", n.hz, n.at, worst)
		}
	}
}

func BenchmarkMusic(b *testing.B) {
	for i := 0; i < b.N; i++ {
		music(false)
		music(true)
	}
}
