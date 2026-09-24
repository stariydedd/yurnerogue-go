// Команда game — точка входа YurneROGUE: нативное окно и WASM-сборка.
package main

import (
	"log"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/stariydedd/yurnerogue-go/internal/game"
	"github.com/stariydedd/yurnerogue-go/internal/locale"
	"github.com/stariydedd/yurnerogue-go/internal/render"
)

func main() {
	layout := chooseLayout()
	layout.Language = locale.Load()

	renderer, err := render.New(layout)
	if err != nil {
		log.Fatal(err)
	}

	ebiten.SetWindowSize(layout.ScreenW, layout.ScreenH)
	if !layout.Touch {
		ebiten.SetWindowSize(1280, 720)
		ebiten.SetWindowResizingMode(ebiten.WindowResizingModeEnabled)
	}
	ebiten.SetWindowTitle("YurneROGUE")
	g := game.New(renderer)
	g.EnableAudio()
	if err := ebiten.RunGame(g); err != nil {
		log.Fatal(err)
	}
}
