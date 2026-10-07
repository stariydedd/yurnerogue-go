//go:build js

package game

import (
	"syscall/js"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/stariydedd/yurnerogue-go/internal/render"
)

// Настоящий DOM input получает исходное касание, поэтому Safari может открыть клавиатуру.
// Опрос значений избавляет от Go-колбэков, выполняющихся внутри обработчика событий браузера.
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
	bridge.Call("show", g.nameInput, box.Min.X, box.Min.Y, box.Dx(), box.Dy(), l.ScreenW, l.ScreenH, string(l.Language))
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
