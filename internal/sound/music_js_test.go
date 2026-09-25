//go:build js

package sound

import (
	"encoding/binary"
	"syscall/js"
	"testing"
)

// A stand-in for the browser AudioContext, recording what the track does.
const fakeAudioContext = `
globalThis.AudioContext = class {
	constructor(options) {
		this.sampleRate = options.sampleRate;
		this.state = "suspended";
		this.destination = {};
		globalThis.fakeAudio = this;
	}
	createBuffer(channels, frames, rate) {
		this.buffer = {channels, frames, rate, data: []};
		this.buffer.copyToChannel = (samples, channel) => { this.buffer.data[channel] = Array.from(samples); };
		return this.buffer;
	}
	createBufferSource() {
		this.source = {connect() {}, start() { this.started = true; }};
		return this.source;
	}
	createGain() {
		this.gainNode = {gain: {value: 1}, connect() {}};
		return this.gainNode;
	}
	resume() { this.state = "running"; }
	suspend() { this.state = "suspended"; }
};`

func TestMusicLoopsInWebAudio(t *testing.T) {
	js.Global().Get("Function").New(fakeAudioContext).Invoke()
	defer js.Global().Delete("AudioContext")
	webAudio = js.Undefined()
	defer func() { webAudio = js.Undefined() }()

	pcm := make([]byte, 8)
	for i, s := range []int16{1000, -1000, 32767, -32768} {
		binary.LittleEndian.PutUint16(pcm[i*2:], uint16(s))
	}
	track := newMusicTrack(nil, pcm)
	fake := js.Global().Get("fakeAudio")
	buffer, source, gain := fake.Get("buffer"), fake.Get("source"), fake.Get("gainNode").Get("gain")
	if fake.Get("sampleRate").Int() != SampleRate || buffer.Get("channels").Int() != 2 || buffer.Get("frames").Int() != 2 {
		t.Fatal("music buffer does not match the PCM")
	}
	if buffer.Get("data").Index(1).Index(1).Float() != -1 || buffer.Get("data").Index(0).Index(0).Float() != 1000.0/32768 {
		t.Fatal("channels were not copied")
	}
	if !source.Get("loop").Bool() || !source.Get("started").Bool() || gain.Get("value").Float() != 0 {
		t.Fatal("music must start as a silent loop")
	}
	track.SetVolume(0.25)
	if gain.Get("value").Float() != 0.25 {
		t.Fatal("volume did not reach the gain node")
	}
	track.Play()
	if !track.IsPlaying() || fake.Get("state").String() != "running" {
		t.Fatal("play did not resume the context")
	}
	track.Pause()
	if track.IsPlaying() || fake.Get("state").String() != "suspended" {
		t.Fatal("pause did not suspend the context")
	}
}
