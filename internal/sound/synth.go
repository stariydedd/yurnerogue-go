// Package sound mixes synthesized fantasy audio with embedded attack recordings.
package sound

import (
	"encoding/binary"
	"math"
)

const SampleRate = 44100

// Recorded reports whether a cue plays an embedded recording, not synthesis.
func Recorded(c Cue) bool {
	return c == Hit || c == Swing || c == Parry || c == Critical
}

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
	Sharpen
	Parry
	Critical
	cueCount
)

// toneResync is how often, in samples, tone sets its oscillators and decays
// exactly again.
const toneResync = 1024

// tone adds one note to dst: a sine with two overtones under an envelope.
// Every oscillator turns by a fixed rotation and every decay shrinks by a
// fixed factor per sample, instead of calling math.Sin and math.Exp for each
// one; they are set exactly again every toneResync samples, so rounding never
// builds up. This is what lets the music be synthesized at the full rate.
//
// All synthesis uses local deterministic state, never the game's random stream.
func tone(dst []float64, at, duration, hz, gain, attack float64, bell bool) {
	ratios, weights, decays := [3]float64{1, 2, 3}, [3]float64{1, 0.22, 0.08}, [3]float64{}
	if bell {
		ratios, weights, decays = [3]float64{1, 2.01, 3.98}, [3]float64{1, 0.28, 0.09}, [3]float64{0, 3, 5}
	}
	const dt = 1.0 / SampleRate
	type oscillator struct{ sin, cos, turnSin, turnCos, amp, fall float64 }
	var osc [3]oscillator
	for k := range osc {
		osc[k].turnSin, osc[k].turnCos = math.Sincos(2 * math.Pi * hz * ratios[k] * dt)
		osc[k].fall = math.Exp(-decays[k] * dt)
	}
	body, fall := 0.0, math.Exp(-3/duration*dt)
	start := int(at * SampleRate)
	for i := 0; i < int(duration*SampleRate); i++ {
		t := float64(i) / SampleRate
		if i%toneResync == 0 {
			for k := range osc {
				osc[k].sin, osc[k].cos = math.Sincos(2 * math.Pi * hz * ratios[k] * t)
				osc[k].amp = weights[k] * math.Exp(-decays[k]*t)
			}
			body = math.Exp(-3 * t / duration)
		}
		x := 0.0
		for k := range osc {
			o := &osc[k]
			x += o.sin * o.amp
			o.sin, o.cos = o.sin*o.turnCos+o.cos*o.turnSin, o.cos*o.turnCos-o.sin*o.turnSin
			o.amp *= o.fall
		}
		env := body * min(1, t/attack) * min(1, (duration-t)/0.08)
		body *= fall
		dst[(start+i)%len(dst)] += x * env * gain
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
	if Recorded(c) {
		return nil // The engine plays an embedded recording instead.
	}
	duration := 0.7
	if c == Victory {
		return effectPCM(victoryFanfare(), c, 0.12)
	}
	if c == Portal || c == Death || c == Start {
		duration = 2.4
	}
	dst := make([]float64, int(duration*SampleRate))
	if c == Sharpen {
		whetstone(dst)
		return effectPCM(dst, c, 0.10)
	}
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
		case Start:
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

// whetstone: two quick scrapes of a blade on stone, then a rising metallic
// ring, so sharpening never sounds like equipping or reading a scroll.
func whetstone(dst []float64) {
	seed := uint32(4217)
	for _, at := range []float64{0, 0.13} {
		start := int(at * SampleRate)
		last := 0.0
		for i := 0; i < SampleRate*9/100 && start+i < len(dst); i++ {
			seed = seed*1664525 + 1013904223
			n := float64(seed>>8)/8388608 - 1
			last = n - 0.55*last // brighter, hissing noise for the scrape
			t := float64(i) / SampleRate
			env := smoothAttack(t, 0.012) * math.Exp(-26*t)
			dst[start+i] += last * env * 0.32
		}
	}
	tone(dst, 0.24, 0.42, 987.77, 0.16, 0.006, true)
	tone(dst, 0.31, 0.38, 1318.51, 0.13, 0.006, true)
}

// victoryFanfare is the finale of a won run, about four seconds in D major,
// the key of the forest music: a three-note call, a leap to the fifth, a
// rising answer and a long high tonic over a brass chord, bass and a few
// bell sparkles.
func victoryFanfare() []float64 {
	const (
		d3, d4, fs4, a4 = 146.83, 293.66, 369.99, 440.0
		d5, fs5, a5, d6 = 587.33, 739.99, 880.0, 1174.66
		fs6, a6         = 1479.98, 1760.0
		final           = 1.45
	)
	dst := make([]float64, int(4.4*SampleRate))
	// Call and answer, with a brass tone rather than a bell.
	for _, n := range []struct{ at, dur, hz float64 }{
		{0, .12, d5}, {.14, .12, d5}, {.28, .12, d5}, {.42, .6, a5},
		{1.05, .18, fs5}, {1.25, .18, a5}, {final, 2.8, d6},
	} {
		tone(dst, n.at, n.dur, n.hz, .15, .012, false)
	}
	// The chord and bass hold the final note.
	tone(dst, 0, .9, d3, .10, .02, false)
	for _, hz := range []float64{d4, fs4, a4} {
		tone(dst, final, 2.8, hz, .06, .03, false)
	}
	tone(dst, final, 2.8, d3, .10, .03, false)
	// Sparkles over the last chord.
	for i, hz := range []float64{d6, fs6, a6} {
		tone(dst, final+.08+float64(i)*.09, 1.2, hz, .04, .004, true)
	}
	return dst
}
