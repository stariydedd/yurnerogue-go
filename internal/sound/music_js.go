//go:build js

package sound

import (
	"syscall/js"
	"unsafe"

	"github.com/hajimehoshi/ebiten/v2/audio"
)

// In the browser Go runs on the main thread, and so does the Ebitengine
// mixer. A long frame on a slow machine starved it and cut the music, which
// never stops playing. Music is handed to Web Audio instead: the browser loops
// it on its own audio thread, and Go only sets volumes. Short effects stay on
// Ebitengine, where they finish before a frame can starve them.

var webAudio js.Value

// webAudioContext returns the shared context, created on first use. Browsers
// start it suspended until the player interacts with the page.
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

// webAudioPaused is set while the game is unfocused, so a click on the page
// does not resume music the game has paused.
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

// float32Array copies samples into a new JavaScript Float32Array.
func float32Array(samples []float32) js.Value {
	bytes := unsafe.Slice((*byte)(unsafe.Pointer(unsafe.SliceData(samples))), len(samples)*4)
	u8 := js.Global().Get("Uint8Array").New(len(bytes))
	js.CopyBytesToJS(u8, bytes)
	return js.Global().Get("Float32Array").New(u8.Get("buffer"))
}

func (t *webTrack) Play() {
	t.playing = true
	webAudioPaused = false
	webAudio.Call("resume") // allowed once the page had any interaction
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

// SetVolume runs every frame during a crossfade; it touches JavaScript only
// when the value changes.
func (t *webTrack) SetVolume(v float64) {
	if v == t.volume {
		return
	}
	t.volume = v
	t.gain.Get("gain").Set("value", v)
}
