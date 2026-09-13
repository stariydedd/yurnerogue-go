package sound

import (
	"encoding/binary"
	"math"
)

// Apply tone shaping AFTER peak normalization: normalizing again would boost
// the softened details back up. Music keeps its own, unchanged mastering path.
func effectPCM(samples []float64, c Cue, echo float64) []byte {
	data := pcmWithEcho(samples, false, echo)
	softenEffect(data, c)
	return data
}

func effectProfile(c Cue) (cutoff, gain float64) {
	switch c {
	case Hurt, Kill:
		return 2100, 0.64
	case Click:
		return 1900, 0.42
	case StepGrass, StepTrail:
		return 2400, 0.32
	default:
		return 2700, 0.70
	}
}

func softenEffect(data []byte, c Cue) {
	cutoff, gain := effectProfile(c)
	alpha := 1 - math.Exp(-2*math.Pi*cutoff/SampleRate)
	var low, smooth [2]float64
	frames := len(data) / 4
	for i := 0; i < frames; i++ {
		// Smooth edges without introducing pre-echo before delayed impacts.
		fade := smoothAttack(float64(i)/SampleRate, 0.008)
		fade *= smoothAttack(float64(frames-1-i)/SampleRate, 0.012)
		for channel := 0; channel < 2; channel++ {
			offset := i*4 + channel*2
			v := float64(int16(binary.LittleEndian.Uint16(data[offset:])))
			low[channel] += alpha * (v - low[channel])
			smooth[channel] += alpha * (low[channel] - smooth[channel])
			binary.LittleEndian.PutUint16(data[offset:], uint16(int16(smooth[channel]*gain*fade)))
		}
	}
}

func smoothAttack(t, duration float64) float64 {
	u := math.Max(0, math.Min(1, t/duration))
	return u * u * (3 - 2*u)
}
