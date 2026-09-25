package sound

import (
	"encoding/binary"
	"log"
	"math"
	"time"

	"github.com/hajimehoshi/ebiten/v2/audio"
)

type Settings struct{ Music, Effects int }

// Reserve mixer headroom even with every voice at full volume.
const masterGain = 0.85
const musicGain, effectsGain = 0.30 * masterGain, 0.20 * masterGain
const maxVoices = 4

// Player.Play fills the whole player buffer on the game thread before it
// returns, 0.5 s by default. The browser audio worklet keeps only about 46 ms
// at 44.1 kHz, so on a slow machine that fill starved the mix and cut the
// music at every effect. Effects start with a short buffer instead; the mixer
// tops it up in the background.
const voiceBuffer = 100 * time.Millisecond

func Defaults() Settings { return Settings{Music: 25, Effects: 75} }
func (s Settings) Valid() bool {
	return s.Music >= 0 && s.Music <= 100 && s.Effects >= 0 && s.Effects <= 100
}
func NextVolume(v int) int { return (v/25*25 + 25) % 125 }

type bank struct {
	menu, world []byte
	cues        [cueCount][]byte
	steps       [2][stepVariants][]byte
	attacks     [attackVariants][]byte
	misses      [missVariants][]byte
}

// toFloat32 converts every effect clip once at load, so starting a voice
// copies ready float32 samples instead of converting 16-bit PCM on the game
// thread. Music keeps 16-bit: it loops from the start and is twice as large.
func (b *bank) toFloat32() {
	convert := func(clip *[]byte) { *clip = float32PCM(*clip) }
	for c := range b.cues {
		convert(&b.cues[c])
	}
	for surface := range b.steps {
		for variant := range b.steps[surface] {
			convert(&b.steps[surface][variant])
		}
	}
	for i := range b.attacks {
		convert(&b.attacks[i])
	}
	for i := range b.misses {
		convert(&b.misses[i])
	}
}

// float32PCM turns 16-bit little-endian PCM into float32 little-endian PCM.
func float32PCM(pcm []byte) []byte {
	out := make([]byte, len(pcm)/2*4)
	for i := 0; i+1 < len(pcm); i += 2 {
		v := float32(int16(binary.LittleEndian.Uint16(pcm[i:]))) / (1 << 15)
		binary.LittleEndian.PutUint32(out[i*2:], math.Float32bits(v))
	}
	return out
}

type Engine struct {
	context      *audio.Context
	ready        chan bank
	bank         *bank
	tracks       [2]musicTrack
	voices       []*audio.Player
	settings     Settings
	mix          [2]float64
	last         [cueCount]int
	tick         int
	stepBags     [2]variantBag
	attackBag    variantBag
	missBag      variantBag
	lastStepTick int
	stepPlayed   bool
}

func New() *Engine {
	e := &Engine{context: audio.NewContext(SampleRate), ready: make(chan bank, 1), settings: loadSettings()}
	go func() {
		b := bank{menu: music(false), world: music(true)}
		for c := Cue(0); c < cueCount; c++ {
			if Recorded(c) {
				continue
			}
			b.cues[c] = effect(c)
		}
		var err error
		for c, clip := range abilityRecordings {
			if b.cues[c], err = loadRecording(clip.name, clip.gain); err != nil {
				log.Printf("Ability recording unavailable: %v", err)
			}
		}
		b.attacks, err = loadAttackSamples()
		if err != nil {
			log.Printf("Attack recordings unavailable: %v", err)
		}
		b.misses, err = loadMissSamples()
		if err != nil {
			log.Printf("Miss recordings unavailable: %v", err)
		}
		for surface := range b.steps {
			for variant := range b.steps[surface] {
				b.steps[surface][variant] = footstep(StepGrass+Cue(surface), variant)
			}
		}
		b.toFloat32()
		e.ready <- b
	}()
	return e
}

func (e *Engine) Settings() Settings {
	if e == nil {
		return Defaults()
	}
	return e.settings
}
func (e *Engine) Adjust(music bool) {
	settings := e.Settings()
	value := settings.Effects
	if music {
		value = settings.Music
	}
	e.SetVolume(music, NextVolume(value))
}

func (e *Engine) SetVolume(music bool, value int) {
	if e == nil {
		return
	}
	value = max(0, min(100, value))
	before := e.settings
	if music {
		e.settings.Music = value
	} else {
		e.settings.Effects = value
	}
	if e.settings == before {
		return
	}
	saveSettings(e.settings)
	for _, p := range e.voices {
		p.SetVolume(float64(e.settings.Effects) / 100 * effectsGain)
	}
}

// Update runs on the game thread; generation does not touch players or settings.
func (e *Engine) Update(exploring, focused bool) {
	if e == nil {
		return
	}
	e.tick++
	if e.bank == nil {
		select {
		case b := <-e.ready:
			e.bank = &b
			for i, data := range [][]byte{b.menu, b.world} {
				if p := newMusicTrack(e.context, data); p != nil {
					e.tracks[i] = p
					p.SetVolume(0)
				}
			}
			// The tracks own their samples now; the bank copy would only weigh
			// on the garbage collector.
			e.bank.menu, e.bank.world = nil, nil
		default:
			return
		}
	}
	for i, p := range e.tracks {
		if p == nil {
			continue
		}
		if !focused {
			p.Pause()
			continue
		}
		if !p.IsPlaying() {
			p.Play()
		}
		target := 0.0
		if (i == 1) == exploring {
			target = float64(e.settings.Music) / 100
		}
		e.mix[i] += math.Max(-0.005, math.Min(0.005, target-e.mix[i]))
		if e.settings.Music == 0 {
			e.mix[i] = 0
		}
		p.SetVolume(e.mix[i] * musicGain)
	}
	kept := e.voices[:0]
	for _, p := range e.voices {
		if !focused || !p.IsPlaying() {
			p.Close()
		} else {
			kept = append(kept, p)
		}
	}
	e.voices = kept
}

func (e *Engine) Play(c Cue) {
	if e == nil || e.bank == nil || e.settings.Effects == 0 || c < 0 || c >= cueCount {
		return
	}
	if e.last[c] != 0 && e.tick-e.last[c] < 6 {
		return
	}
	data := e.bank.cues[c]
	if c == Critical {
		// A Critical Strike is a regular attack recording (same no-repeat bag
		// as ordinary hits) with Blade Dance layered on top.
		e.last[c] = e.tick
		e.startVoice(e.bank.attacks[e.attackBag.next(attackVariants)])
		e.startVoice(data)
		return
	}
	if c == Hit {
		data = e.bank.attacks[e.attackBag.next(attackVariants)]
	}
	if c == Swing {
		data = e.bank.misses[e.missBag.next(missVariants)]
	}
	if c == StepGrass || c == StepTrail {
		// One cadence gate for both surfaces; skipped sounds don't consume a variant.
		if e.stepPlayed && e.tick-e.lastStepTick < 6 {
			return
		}
		surface := int(c - StepGrass)
		data = e.bank.steps[surface][e.stepBags[surface].next(stepVariants)]
		e.lastStepTick, e.stepPlayed = e.tick, true
	}
	if len(data) == 0 {
		return
	}
	e.last[c] = e.tick
	e.startVoice(data)
}

func (e *Engine) startVoice(data []byte) {
	if len(data) == 0 {
		return
	}
	if len(e.voices) >= maxVoices {
		e.voices[0].Close()
		e.voices = e.voices[1:]
	}
	p := e.context.NewPlayerF32FromBytes(data)
	p.SetBufferSize(voiceBuffer)
	p.SetVolume(float64(e.settings.Effects) / 100 * effectsGain)
	p.Play()
	e.voices = append(e.voices, p)
}
