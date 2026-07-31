// Команда game — точка входа YurneROGUE: нативное окно и WASM-сборка.
package main

import (
	"log"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/stariydedd/yurnerogue-go/internal/game"
	"github.com/stariydedd/yurnerogue-go/internal/render"
)

func main() {
	layout := chooseLayout()

	renderer, err := render.New(layout)
	if err != nil {
		log.Fatal(err)
	}

	ebiten.SetWindowSize(layout.ScreenW, layout.ScreenH)
	ebiten.SetWindowTitle("YurneROGUE")
	if err := ebiten.RunGame(game.New(renderer)); err != nil {
		log.Fatal(err)
	}
}
