package sound

import "math"

const stepVariants = 4

// Each sound family has its own shuffle bag, independent of simulation randomness.
// The boundary between bags also excludes the last played variant.
type variantBag struct {
	bag             [stepVariants]int
	remaining, last int
	used            bool
	seed            uint32
}

func (b *variantBag) random(n int) int {
	b.seed = b.seed*1664525 + 1013904223
	return int(b.seed>>8) % n
}

func (b *variantBag) next(count int) int {
	if count < 2 || count > len(b.bag) {
		panic("invalid sound variant count")
	}
	if b.remaining == 0 {
		for i := 0; i < count; i++ {
			b.bag[i] = i
		}
		for i := count - 1; i > 0; i-- {
			j := b.random(i + 1)
			b.bag[i], b.bag[j] = b.bag[j], b.bag[i]
		}
		if b.used && b.bag[count-1] == b.last {
			j := b.random(count - 1)
			b.bag[j], b.bag[count-1] = b.bag[count-1], b.bag[j]
		}
		b.remaining = count
	}
	b.remaining--
	b.last, b.used = b.bag[b.remaining], true
	return b.last
}

// Heel, sole and a short scuff form one step; no long reverberation on grass
// or outdoor trails. Variants change noise, weight and the sole's timing.
func footstep(c Cue, variant int) []byte {
	const duration = 0.24
	dst := make([]float64, int(duration*SampleRate))
	seed := uint32(19073 + int(c)*733 + variant*3517)
	low, smooth := 0.0, 0.0
	soleAt := 0.032 + float64(variant)*0.005
	for i := range dst {
		seed = seed*1664525 + 1013904223
		noise := float64(seed>>8)/8388608 - 1
		low += 0.08 * (noise - low)
		smooth += 0.48 * (noise - smooth)
		t := float64(i) / SampleRate
		heel := smoothAttack(t, 0.012) * math.Exp(-45*t)
		sole := 0.0
		if t > soleAt {
			u := t - soleAt
			sole = smoothAttack(u, 0.014) * math.Exp(-35*u)
		}
		weight := math.Sin(2*math.Pi*float64(78+variant*9)*t) * heel * 0.12
		if c == StepGrass {
			// Soft compression with a longer, irregular foliage rustle.
			rustle := (smooth - low) * (0.55 + 0.3*math.Sin(2*math.Pi*float64(31+variant*5)*t))
			dst[i] = weight + low*heel*0.9 + rustle*(heel*0.18+sole*0.55)
		} else {
			// A firm contact followed by the scrape of grit under the sole.
			grit := (noise - smooth) * (heel*0.65 + sole*0.22)
			dst[i] = weight*1.4 + smooth*heel*0.5 + grit
		}
	}
	return effectPCM(dst, c, 0)
}
