// Package sound mixes synthesized fantasy audio with embedded attack recordings.
package sound

import (
	"encoding/binary"
	"math"
)

const SampleRate = 22050

type Cue int

const (
	Click Cue = iota
	Start
	Swing
	Hit
	Hurt
	Kill
	Pickup
	Heal
	Clarity
	Scroll
	Equip
	Portal
	Death
	Victory
	StepGrass
	StepTrail
	cueCount
)

// All synthesis uses local deterministic state, never the game's random stream.
func tone(dst []float64, at, duration, hz, gain, attack float64, bell bool) {
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

// Music is a 32-second seamless bed of warm drones, harp-like notes and echoes.
// The two related arrangements crossfade without an abrupt change of key.
func music(exploring bool) []byte {
	dst := make([]float64, 32*SampleRate)
	roots := []float64{146.8324, 130.8128, 116.5409, 130.8128}
	melody := []float64{293.6648, 349.2282, 440, 391.9954, 261.6256, 349.2282, 293.6648, 220}
	for bar, root := range roots {
		for _, ratio := range []float64{0.5, 1, 1.5} {
			tone(dst, float64(bar*8), 12, root*ratio, 0.10, 2, false)
		}
		for n := 0; n < 4; n++ {
			hz := melody[(bar*2+n)%len(melody)]
			gain := 0.11
			if exploring {
				hz *= 0.5
				gain = 0.08
			}
			tone(dst, float64(bar*8+n*2)+0.5, 4, hz, gain, 0.025, true)
		}
		if exploring {
			tone(dst, float64(bar*8), 7, root*0.25, 0.10, 0.7, false)
		}
	}
	return pcm(dst, true)
}

func effect(c Cue) []byte {
	if c == StepGrass || c == StepTrail {
		return footstep(c, 0)
	}
	if c == Hit || c == Swing {
		return nil // The engine selects an embedded attack or miss recording.
	}
	duration := 0.7
	if c == Portal || c == Death || c == Victory || c == Start {
		duration = 2.4
	}
	dst := make([]float64, int(duration*SampleRate))
	if c == Hurt || c == Kill || c == Equip {
		seed := uint32(913 + c)
		last := 0.0
		for i := range dst {
			seed = seed*1664525 + 1013904223
			n := float64(seed>>8)/8388608 - 1
			last = 0.7*last + 0.3*n
			t := float64(i) / SampleRate
			env := smoothAttack(t, 0.020) * math.Exp(-14*t) * math.Min(1, (duration-t)/0.04)
			dst[i] = last * env * 0.65
		}
		hz := 100.0
		if c == Equip {
			hz = 740
		}
		tone(dst, 0, 0.35, hz, 0.20, 0.018, c == Equip)
	} else {
		notes := []float64{587.33, 880}
		switch c {
		case Click:
			notes = []float64{440}
			duration = 0.10
		case Start, Victory:
			notes = []float64{293.66, 349.23, 440, 587.33}
		case Heal:
			notes = []float64{349.23, 440, 523.25}
		case Clarity:
			notes = []float64{440, 659.25, 880}
		case Scroll:
			notes = []float64{293.66, 440, 659.25}
		case Portal:
			notes = []float64{146.83, 220, 293.66, 440, 587.33}
		case Death:
			notes = []float64{220, 174.61, 146.83, 73.42}
		}
		for i, hz := range notes {
			tone(dst, float64(i)*0.10, math.Min(duration-float64(i)*0.10, 1.8), hz, 0.18, 0.024, true)
		}
	}
	return effectPCM(dst, c, 0.12)
}

func pcm(samples []float64, loop bool) []byte {
	return pcmWithEcho(samples, loop, 0.22)
}

func pcmWithEcho(samples []float64, loop bool, echo float64) []byte {
	peak := 0.0
	for _, v := range samples {
		peak = math.Max(peak, math.Abs(v))
	}
	scale := 0.55 / math.Max(0.001, peak)
	result := make([]byte, len(samples)*4)
	for i, v := range samples {
		for channel, delay := range []int{23 * SampleRate / 100, 37 * SampleRate / 100} {
			j := i - delay
			if loop && j < 0 {
				j += len(samples)
			}
			x := v
			if j >= 0 {
				x += echo * samples[j]
			}
			if !loop {
				x *= math.Min(1, float64(len(samples)-1-i)/(SampleRate*0.08))
			}
			x = math.Max(-0.8, math.Min(0.8, x*scale))
			binary.LittleEndian.PutUint16(result[i*4+channel*2:], uint16(int16(x*32767)))
		}
	}
	return result
}
