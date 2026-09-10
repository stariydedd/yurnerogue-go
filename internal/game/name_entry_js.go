//go:build js

package game

import (
	"syscall/js"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/stariydedd/yurnerogue-go/internal/render"
)

// A real DOM input receives the original tap, so Safari can open its keyboard.
// Polling values avoids Go callbacks running inside a browser event handler.
func (g *Game) syncBrowserNameEntry() bool {
	bridge := js.Global().Get("yurneNameEntry")
	if !bridge.Truthy() {
		return false
	}
	l := g.renderer.Layout
	if !l.Touch || g.state != StateNameEntry {
		bridge.Call("hide")
		return false
	}
	box := render.NameEntryBounds(l).Inset(6)
	bridge.Call("show", g.nameInput, box.Min.X, box.Min.Y, box.Dx(), box.Dy(), l.ScreenW, l.ScreenH)
	value := bridge.Call("read")
	g.nameInput = value.Get("value").String()
	switch value.Get("action").String() {
	case "start":
		g.HandleKey(ebiten.KeyEnter)
	case "back":
		g.HandleKey(ebiten.KeyEscape)
	}
	return true
}
