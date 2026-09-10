const assert = require("node:assert/strict");
const fs = require("node:fs");
const path = require("node:path");
const vm = require("node:vm");
const { test } = require("node:test");
const html = fs.readFileSync(path.join(__dirname, "index.html"), "utf8");
const script = html.match(/<script id="name-entry">([\s\S]*?)<\/script>/)[1];

function setup() {
    const events = {};
    const input = {
        hidden: true, value: "", style: {},
        addEventListener(name, handler) { events[name] = handler; },
        blur() { events.blur?.(); },
    };
    const window = {};
    vm.runInNewContext(script, {window, document: {getElementById: () => input}});
    const bridge = window.yurneNameEntry;
    const show = name => bridge.show(name, 18, 426, 444, 36, 480, 860);
    return {input, events, bridge, show};
}

test("real name input stays aligned and does not reset while typing", () => {
    assert.match(html, /<input id="player-name" type="text"/);
    const {input, bridge, events, show} = setup();
    show("old");
    assert.equal(input.hidden, false);
    assert.equal(input.style.left, "3.75%");
    input.value = "Новый ник";
    events.input();
    show("old");
    assert.equal(input.value, "Новый ник");
    assert.equal(bridge.read().value, "Новый ник");
    bridge.hide();
    assert.equal(input.hidden, true);
    show("next");
    assert.equal(input.value, "next");
});

test("composition, pasted controls and Unicode length are handled", () => {
    const {input, bridge, events, show} = setup();
    show("");
    events.compositionstart();
    input.value = "临时";
    events.input();
    assert.equal(bridge.read().value, "");
    events.compositionend();
    assert.equal(bridge.read().value, "临时");
    input.value = "\n" + "😀".repeat(20);
    events.input();
    assert.equal(Array.from(bridge.read().value).length, 16);
    assert.equal(input.value, "😀".repeat(16));
});

test("keyboard submits or cancels once without leaking keys into the game", () => {
    const {input, bridge, events, show} = setup();
    show("");
    input.value = "tester";
    for (const [key, action] of [["Enter", "start"], ["Escape", "back"]]) {
        let stopped = false, prevented = false;
        events.keydown({key, stopPropagation() { stopped = true; }, preventDefault() { prevented = true; }});
        assert.equal(stopped, true);
        assert.equal(prevented, true);
        const result = bridge.read();
        assert.equal(result.value, "tester");
        assert.equal(result.action, action);
        assert.equal(bridge.read().action, "");
    }
});
