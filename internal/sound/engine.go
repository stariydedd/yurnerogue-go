package sound

import (
	"encoding/binary"
	"log"
	"math"
	"time"

	"github.com/hajimehoshi/ebiten/v2/audio"
)

type Settings struct{ Music, Effects int }

// Оставляем запас микшеру даже при всех голосах на полной громкости.
const masterGain = 0.85
const musicGain, effectsGain = 0.30 * masterGain, 0.20 * masterGain
const maxVoices = 4

// Player.Play заполняет весь буфер плеера в игровом потоке, прежде чем вернуться,
// по умолчанию 0,5 с. Звуковой worklet браузера держит лишь около 46 мс
// при 44,1 кГц, поэтому на медленной машине это заполнение морило микс голодом
// и обрывало музыку на каждом эффекте. Эффекты стартуют с коротким буфером;
// микшер доливает его в фоне.
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

// toFloat32 один раз при загрузке переводит каждый клип эффекта, чтобы запуск голоса
// копировал готовые float32-отсчёты, а не переводил 16-битный PCM в игровом
// потоке. Музыка остаётся 16-битной: она играет по кругу и вдвое больше.
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

// float32PCM превращает 16-битный little-endian PCM в float32 little-endian PCM.
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
	silent       bool
	// quietUntil не пускает музыку, пока не закончится фанфара победы или мелодия
	// смерти, даже если игрок раньше ушёл с экрана итогов.
	quietUntil int
}

// SilenceMusic быстро убирает музыку и держит её выключенной на экране итогов
// забега; false даёт ей вернуться с обычной скоростью.
func (e *Engine) SilenceMusic(on bool) {
	if e != nil {
		e.silent = on
	}
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

// Update выполняется в игровом потоке; генерация не трогает плееры и настройки.
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
			// Теперь сэмплы принадлежат трекам; копия в банке только нагружала бы
			// сборщик мусора.
			e.bank.menu, e.bank.world = nil, nil
		default:
			return
		}
	}
	if !focused {
		e.quietUntil = 0 // без фокуса голоса закрыты: фанфары больше нет
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
		target, step := 0.0, 0.005
		if (i == 1) == exploring {
			target = float64(e.settings.Music) / 100
		}
		if e.silent || e.tick < e.quietUntil {
			target, step = 0, 0.05 // затихает примерно за треть секунды
		}
		e.mix[i] += math.Max(-step, math.Min(step, target-e.mix[i]))
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
	if (c == Victory || c == Death) && len(e.bank.cues[c]) > 0 {
		e.quietFor(c) // только для звука, который действительно запускается
	}
	data := e.bank.cues[c]
	if c == Critical {
		// Критический удар: обычная запись атаки (тот же мешок без повторов,
		// что у обычных ударов) с наложенным Blade Dance.
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
		// Один ограничитель темпа для обеих поверхностей; пропущенные звуки не тратят вариант.
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

// quietFor не пускает музыку, пока звучит звук (стерео float32,
// 60 тиков в секунду).
func (e *Engine) quietFor(c Cue) {
	frames := len(e.bank.cues[c]) / 8
	e.quietUntil = e.tick + frames*60/SampleRate + 1
}
