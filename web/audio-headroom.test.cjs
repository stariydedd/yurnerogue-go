const assert = require("node:assert/strict");
const fs = require("node:fs");
const path = require("node:path");
const vm = require("node:vm");
const { test } = require("node:test");
const html = fs.readFileSync(path.join(__dirname, "index.html"), "utf8");
const script = html.match(/<script id="audio-headroom">([\s\S]*?)<\/script>/)[1];
const oto = fs.readFileSync(path.join(__dirname, "oto-worklet.fixture.js"), "utf8");

function load(window = {}) {
    vm.runInNewContext(script, {window, fetch: window.fetch, URL: window.URL, Blob: window.Blob});
    return window;
}

// Runs a worklet script and returns one processor with a fake message port.
function processor(source) {
    let Processor;
    const sent = [];
    class AudioWorkletProcessor { constructor() { this.port = {postMessage: (m) => sent.push(m)}; } }
    vm.runInNewContext(source, {AudioWorkletProcessor, Float32Array, registerProcessor: (_, c) => { Processor = c; }});
    const p = new Processor();
    const block = () => [[new Float32Array(128), new Float32Array(128)]];
    return {
        p, sent,
        receive(frames) { p.port.onmessage({data: new Float32Array(frames * 2)}); },
        play(blocks) { for (let i = 0; i < blocks; i++) p.process([], block(), {}); },
    };
}

test("buffer doubles once per underrun, only after sound started, up to 8192 frames", () => {
    const patched = load().yurnePatchOtoWorklet(oto);
    assert.ok(patched);
    const w = processor(patched);
    w.play(10); // empty before the first data: startup, not an underrun
    assert.equal(w.p.bufferSize_, 2048);
    for (const want of [4096, 8192, 8192]) {
        w.receive(2048);
        w.play(16 + 40); // drain 2048 frames, then starve for a while
        assert.equal(w.p.bufferSize_, want);
    }
    assert.ok(w.sent.length > 0, "the worklet keeps requesting data");
});

test("a steady supply never grows the buffer", () => {
    const w = processor(load().yurnePatchOtoWorklet(oto));
    w.receive(2048);
    for (let i = 0; i < 50; i++) {
        w.play(8);
        w.receive(1024);
    }
    assert.equal(w.p.bufferSize_, 2048);
});

test("unknown worklets load untouched", () => {
    const {yurnePatchOtoWorklet} = load();
    assert.equal(yurnePatchOtoWorklet("registerProcessor('other', class {})"), null);
    assert.equal(yurnePatchOtoWorklet(oto.replace("this.buf_ = newBuf;", "this.buf_ = merged;")), null);
});

test("addModule gets the patched oto worklet and other modules as they are", async () => {
    const loaded = [];
    class AudioWorklet {}
    AudioWorklet.prototype.addModule = function (url) { loaded.push(url); return Promise.resolve(); };
    const sources = {"oto.js": oto, "other.js": "registerProcessor('x', class {})"};
    const window = {
        AudioWorklet,
        fetch: (url) => Promise.resolve({text: () => Promise.resolve(sources[url])}),
        URL: {createObjectURL: (blob) => "blob:" + blob.parts[0].length},
        Blob: class { constructor(parts) { this.parts = parts; } },
    };
    load(window);
    await new AudioWorklet().addModule("oto.js");
    await new AudioWorklet().addModule("other.js");
    assert.match(loaded[0], /^blob:/);
    assert.equal(loaded[1], "other.js");
});
