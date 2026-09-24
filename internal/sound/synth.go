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

// musicRate synthesizes the music at half the playback rate: drones and harp
// notes have almost nothing above 5 kHz, so this halves startup work and the
// result is upsampled to SampleRate. Effects keep the full rate for clarity.
const musicRate = SampleRate / 2

// All synthesis uses local deterministic state, never the game's random stream.
func tone(dst []float64, at, duration, hz, gain, attack float64, bell bool) {
	toneAt(dst, SampleRate, at, duration, hz, gain, attack, bell)
}

func toneAt(dst []float64, rate int, at, duration, hz, gain, attack float64, bell bool) {
	for i := 0; i < int(duration*float64(rate)); i++ {
		t := float64(i) / float64(rate)
		env := math.Min(1, t/attack) * math.Exp(-3*t/duration)
		env *= math.Min(1, (duration-t)/0.08)
		x := math.Sin(2 * math.Pi * hz * t)
		if bell {
			x += 0.28 * math.Sin(2*math.Pi*hz*2.01*t) * math.Exp(-3*t)
			x += 0.09 * math.Sin(2*math.Pi*hz*3.98*t) * math.Exp(-5*t)
		} else {
			x += 0.22*math.Sin(2*math.Pi*hz*2*t) + 0.08*math.Sin(2*math.Pi*hz*3*t)
		}
		dst[(int(at*float64(rate))+i)%len(dst)] += x * env * gain
	}
}

// Music is a 32-second seamless bed of warm drones, harp-like notes and echoes.
// The two related arrangements crossfade without an abrupt change of key.
func music(exploring bool) []byte {
	dst := make([]float64, 32*musicRate)
	roots := []float64{146.8324, 130.8128, 116.5409, 130.8128}
	melody := []float64{293.6648, 349.2282, 440, 391.9954, 261.6256, 349.2282, 293.6648, 220}
	for bar, root := range roots {
		for _, ratio := range []float64{0.5, 1, 1.5} {
			toneAt(dst, musicRate, float64(bar*8), 12, root*ratio, 0.10, 2, false)
		}
		for n := 0; n < 4; n++ {
			hz := melody[(bar*2+n)%len(melody)]
			gain := 0.11
			if exploring {
				hz *= 0.5
				gain = 0.08
			}
			toneAt(dst, musicRate, float64(bar*8+n*2)+0.5, 4, hz, gain, 0.025, true)
		}
		if exploring {
			toneAt(dst, musicRate, float64(bar*8), 7, root*0.25, 0.10, 0.7, false)
		}
	}
	return pcm(upsampleLoop(dst, SampleRate/musicRate), true)
}

func effect(c Cue) []byte {
	if c == StepGrass || c == StepTrail {
		return footstep(c, 0)
	}
	if Recorded(c) {
		return nil // The engine plays an embedded recording instead.
	}
	duration := 0.7
	if c == Portal || c == Death || c == Victory || c == Start {
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

// upsampleLoop raises a looping buffer's rate by an integer factor with linear
// interpolation; the last samples interpolate towards the first, keeping the
// loop seamless.
func upsampleLoop(src []float64, factor int) []float64 {
	dst := make([]float64, len(src)*factor)
	for i, v := range src {
		next := src[(i+1)%len(src)]
		for k := 0; k < factor; k++ {
			dst[i*factor+k] = v + (next-v)*float64(k)/float64(factor)
		}
	}
	return dst
}
