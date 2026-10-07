//go:build js

package sound

import (
	"syscall/js"
	"unsafe"

	"github.com/hajimehoshi/ebiten/v2/audio"
)

// В браузере Go работает в главном потоке, и микшер Ebitengine тоже.
// Длинный кадр на медленной машине морил его голодом и обрывал музыку, которая
// играет всегда. Поэтому музыка отдана Web Audio: браузер крутит её
// в своём звуковом потоке, а Go только задаёт громкость. Короткие эффекты
// остаются в Ebitengine: они заканчиваются раньше, чем кадр успеет их уморить.

var webAudio js.Value

// webAudioContext возвращает общий контекст, созданный при первом обращении. Браузеры
// запускают его приостановленным, пока игрок не взаимодействует со страницей.
func webAudioContext() js.Value {
	if !webAudio.IsUndefined() {
		return webAudio
	}
	class := js.Global().Get("AudioContext")
	if !class.Truthy() {
		class = js.Global().Get("webkitAudioContext")
	}
	if !class.Truthy() {
		webAudio = js.Null()
		return webAudio
	}
	options := js.Global().Get("Object").New()
	options.Set("sampleRate", SampleRate)
	webAudio = class.New(options)
	unlock := js.FuncOf(func(js.Value, []js.Value) any {
		if webAudio.Get("state").String() == "suspended" && !webAudioPaused {
			webAudio.Call("resume")
		}
		return nil
	})
	if document := js.Global().Get("document"); document.Truthy() {
		for _, event := range []string{"keydown", "pointerdown", "touchend"} {
			document.Call("addEventListener", event, unlock)
		}
	}
	return webAudio
}

// webAudioPaused выставлен, пока у игры нет фокуса, чтобы щелчок по странице
// не возобновлял музыку, которую игра поставила на паузу.
var webAudioPaused bool

type webTrack struct {
	gain    js.Value
	volume  float64
	playing bool
}

func newMusicTrack(context *audio.Context, pcm []byte) musicTrack {
	ctx := webAudioContext()
	if ctx.IsNull() {
		return ebitenTrack(context, pcm)
	}
	left, right := stereoFloat32(pcm)
	buffer := ctx.Call("createBuffer", 2, len(left), SampleRate)
	for channel, samples := range [][]float32{left, right} {
		buffer.Call("copyToChannel", float32Array(samples), channel)
	}
	source := ctx.Call("createBufferSource")
	source.Set("buffer", buffer)
	source.Set("loop", true)
	gain := ctx.Call("createGain")
	gain.Get("gain").Set("value", 0)
	source.Call("connect", gain)
	gain.Call("connect", ctx.Get("destination"))
	source.Call("start")
	return &webTrack{gain: gain}
}

// float32Array копирует отсчёты в новый JavaScript Float32Array.
func float32Array(samples []float32) js.Value {
	bytes := unsafe.Slice((*byte)(unsafe.Pointer(unsafe.SliceData(samples))), len(samples)*4)
	u8 := js.Global().Get("Uint8Array").New(len(bytes))
	js.CopyBytesToJS(u8, bytes)
	return js.Global().Get("Float32Array").New(u8.Get("buffer"))
}

func (t *webTrack) Play() {
	t.playing = true
	webAudioPaused = false
	webAudio.Call("resume") // разрешено после любого взаимодействия со страницей
}

func (t *webTrack) Pause() {
	if !t.playing {
		return
	}
	t.playing = false
	webAudioPaused = true
	webAudio.Call("suspend")
}

func (t *webTrack) IsPlaying() bool { return t.playing }

// SetVolume вызывается каждый кадр во время перехода; к JavaScript обращается только
// при изменении значения.
func (t *webTrack) SetVolume(v float64) {
	if v == t.volume {
		return
	}
	t.volume = v
	t.gain.Get("gain").Set("value", v)
}
