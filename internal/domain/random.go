package domain

import "math/rand"

// Each seeded session owns its generator. Rendering and other sessions must
// never advance it; the server replays exactly the same random decisions.
type randomizer interface {
	Intn(int) int
	Float64() float64
	Shuffle(int, func(int, int))
}

type globalRandom struct{}

func (globalRandom) Intn(n int) int                     { return rand.Intn(n) }
func (globalRandom) Float64() float64                   { return rand.Float64() }
func (globalRandom) Shuffle(n int, swap func(int, int)) { rand.Shuffle(n, swap) }

func random(rng *rand.Rand) randomizer {
	if rng != nil {
		return rng
	}
	return globalRandom{}
}

func source(rng []*rand.Rand) *rand.Rand {
	if len(rng) > 0 {
		return rng[0]
	}
	return nil
}
