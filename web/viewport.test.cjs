const assert = require("node:assert/strict");
const fs = require("node:fs");
const path = require("node:path");
const vm = require("node:vm");
const { test } = require("node:test");

const html = fs.readFileSync(path.join(__dirname, "index.html"), "utf8");
const script = html.match(/<script id="viewport-sizing">([\s\S]*?)<\/script>/)[1];

function start(visibleHeight) {
    const properties = {};
    const events = {};
    const viewportEvents = {};
    const window = {
        innerHeight: 844,
        addEventListener: (name, handler) => { events[name] = handler; },
    };
    if (visibleHeight !== undefined) {
        window.visualViewport = {
            height: visibleHeight,
            addEventListener: (name, handler) => { viewportEvents[name] = handler; },
        };
    }
    const document = { documentElement: { style: {
        setProperty: (name, value) => { properties[name] = value; },
    } } };
    vm.runInNewContext(script, { window, document });
    return { window, properties, events, viewportEvents };
}

test("uses visible Safari height instead of the larger layout viewport", () => {
    const app = start(664);
    assert.equal(app.properties["--viewport-height"], "664px");
    app.window.visualViewport.height = 744;
    app.viewportEvents.resize();
    assert.equal(app.properties["--viewport-height"], "744px");
    app.window.visualViewport.height = 390;
    app.events.resize();
    assert.equal(app.properties["--viewport-height"], "390px");
    app.window.visualViewport.height = 664;
    app.events.pageshow();
    assert.equal(app.properties["--viewport-height"], "664px");
});

test("falls back to window height without VisualViewport", () => {
    const app = start();
    assert.equal(app.properties["--viewport-height"], "844px");
    app.window.innerHeight = 600;
    app.events.resize();
    assert.equal(app.properties["--viewport-height"], "600px");
    assert.equal(start(0).properties["--viewport-height"], "844px");
});

test("canvas follows the same body dimensions used for input by Ebitengine", () => {
    const body = html.match(/\n    body \{([^}]+)\}/)[1];
    const canvas = html.match(/\n    canvas \{([^}]+)\}/)[1];
    assert.match(body, /height: calc\(var\(--viewport-height, 100vh\) - env\(safe-area-inset-bottom, 0px\)\) !important/);
    assert.match(canvas, /position: absolute !important/);
    assert.match(canvas, /width: 100% !important/);
    assert.match(canvas, /height: 100% !important/);
    assert.doesNotMatch(canvas, /100v[wh]/);
});
