const assert = require("node:assert/strict");
const fs = require("node:fs");
const path = require("node:path");
const vm = require("node:vm");
const { test } = require("node:test");
const html = fs.readFileSync(path.join(__dirname, "index.html"), "utf8");
const script = html.match(/<script id="fullscreen-key">([\s\S]*?)<\/script>/)[1];

function setup() {
    const calls = [];
    let listener, capture;
    const document = {
        fullscreenElement: null,
        documentElement: {requestFullscreen() { calls.push("request"); document.fullscreenElement = this; return Promise.resolve(); }},
        exitFullscreen() { calls.push("exit"); document.fullscreenElement = null; return Promise.resolve(); },
    };
    const window = {addEventListener(name, handler, useCapture) { if (name === "keydown") { listener = handler; capture = useCapture; } }};
    vm.runInNewContext(script, {window, document});
    const press = (key, repeat = false) => {
        const e = {key, code: key, repeat, prevented: false, preventDefault() { this.prevented = true; }};
        listener(e);
        return e;
    };
    return {calls, press, capture, document};
}

test("F11 toggles fullscreen for the whole page before the game sees it", () => {
    const {calls, press, capture, document} = setup();
    assert.equal(capture, true, "must run before the canvas handler cancels the key");
    assert.equal(press("F11").prevented, true);
    assert.equal(document.fullscreenElement, document.documentElement);
    press("F11");
    assert.deepEqual(calls, ["request", "exit"]);
});

test("a held F11 does not flicker and other keys pass through", () => {
    const {calls, press} = setup();
    press("F11");
    press("F11", true);
    press("F11", true);
    assert.equal(press("KeyD").prevented, false);
    assert.deepEqual(calls, ["request"]);
});

test("browsers without the Fullscreen API ignore F11 quietly", () => {
    let listener;
    const window = {addEventListener(_, handler) { listener = handler; }};
    vm.runInNewContext(script, {window, document: {documentElement: {}}});
    assert.doesNotThrow(() => listener({key: "F11", code: "F11", preventDefault() {}}));
});
