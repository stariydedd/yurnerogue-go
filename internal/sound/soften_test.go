package sound

import (
	"encoding/binary"
	"math"
	"testing"
)

func TestEffectFilterReducesTrebleWithoutMutingBody(t *testing.T) {
	measure := func(c Cue, hz float64) float64 {
		data := make([]byte, SampleRate*4/2)
		for i := 0; i < len(data)/4; i++ {
			v := int16(12000 * math.Sin(2*math.Pi*hz*float64(i)/SampleRate))
			for channel := 0; channel < 2; channel++ {
				binary.LittleEndian.PutUint16(data[i*4+channel*2:], uint16(v))
			}
		}
		softenEffect(data, c)
		energy := 0.0
		// Measure steady-state response, not the fade at either endpoint.
		from, to := SampleRate/10, SampleRate*4/10
		for i := from; i < to; i++ {
			v := float64(int16(binary.LittleEndian.Uint16(data[i*4:])))
			energy += v * v
		}
		return math.Sqrt(energy/float64(to-from)) / (12000 / math.Sqrt2)
	}
	for c := Cue(0); c < cueCount; c++ {
		if c == Hit || c == Swing {
			continue
		} // Recordings deliberately bypass this filter.
		_, gain := effectProfile(c)
		body, treble := measure(c, 400), measure(c, 6000)
		if body < gain*0.85 || body > gain*1.01 {
			t.Fatalf("cue %d lost body or was renormalized: %f", c, body)
		}
		if treble > body*0.30 {
			t.Fatalf("cue %d still has excessive high-frequency energy", c)
		}
	}
}

func TestEffectSmoothingKeepsChannelsSeparateAndEdgesQuiet(t *testing.T) {
	data := make([]byte, SampleRate*4/10)
	for i := 0; i < len(data)/4; i++ {
		binary.LittleEndian.PutUint16(data[i*4:], 10000)
	}
	softenEffect(data, Hurt)
	last := int16(0)
	for i := 0; i < len(data)/4; i++ {
		left := int16(binary.LittleEndian.Uint16(data[i*4:]))
		right := int16(binary.LittleEndian.Uint16(data[i*4+2:]))
		if right != 0 || left < 0 || left > 6400 {
			t.Fatal("filter leaked channels or overshot input gain")
		}
		if i < SampleRate*8/1000 && left < last {
			t.Fatal("onset is not a smooth ramp")
		}
		last = left
	}
	if binary.LittleEndian.Uint16(data[:2]) != 0 || last != 0 {
		t.Fatal("filter introduced an endpoint click")
	}
}

func TestEffectProfilesPreserveQuietStepsAndControls(t *testing.T) {
	_, combat := effectProfile(Hurt)
	_, click := effectProfile(Click)
	_, step := effectProfile(StepTrail)
	if click >= combat || step >= click || combat >= 1 {
		t.Fatal("soft mix lost relative level balance")
	}
	for c := Cue(0); c < cueCount; c++ {
		if c == Hit || c == Swing {
			continue
		}
		data := effect(c)
		_, gain := effectProfile(c)
		// Source peak is .55; ordinary effects have at most .12 echo.
		limit := 32767*.55*1.12*gain + 1
		for i := 0; i < len(data); i += 2 {
			if math.Abs(float64(int16(binary.LittleEndian.Uint16(data[i:])))) > limit {
				t.Fatalf("cue %d bypasses soft mastering", c)
			}
		}
	}
}
