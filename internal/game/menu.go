package game

import (
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/stariydedd/yurnerogue-go/internal/render"
	"github.com/stariydedd/yurnerogue-go/internal/sound"
)

func menuPage(state State) (render.MenuPage, bool) {
	switch state {
	case StateMainMenu:
		return render.MenuHome, true
	case StateNameEntry:
		return render.MenuName, true
	case StatePauseMenu:
		return render.MenuPause, true
	case StateHelp, StateLeaderboard, StateStarting, StateDeath, StateWin:
		return render.MenuBack, true
	}
	return 0, false
}

func (g *Game) handleMenuPointer(x, y int) {
	if g.handleAudioPointer(x, y) {
		return
	}
	page, ok := menuPage(g.state)
	if !ok {
		return
	}
	action := render.MenuActionAt(g.renderer.Layout, page, x, y)
	if action == "" {
		return
	}
	if g.state == StatePauseMenu {
		g.activatePauseAction(action)
		return
	}
	if g.state == StateMainMenu {
		for i, option := range render.MainMenuOptions {
			if option.Key == action {
				g.menuSelected = i
				g.menuMessage = ""
				g.HandleKey(ebiten.KeyEnter)
				return
			}
		}
	}
	if action == "start" || g.state == StateDeath || g.state == StateWin {
		g.HandleKey(ebiten.KeyEnter)
	} else if action == "back" {
		g.HandleKey(ebiten.KeyEscape)
	}
}

func (g *Game) handlePauseMenu(key ebiten.Key) {
	buttons := render.MenuButtons(g.renderer.Layout, render.MenuPause)
	switch key {
	case ebiten.KeyEscape, ebiten.KeyQ:
		g.state = StatePlaying
	case ebiten.KeyUp, ebiten.KeyW:
		g.pauseSelected = (g.pauseSelected + len(buttons) - 1) % len(buttons)
		g.audio.Play(sound.Click)
	case ebiten.KeyDown, ebiten.KeyS:
		g.pauseSelected = (g.pauseSelected + 1) % len(buttons)
		g.audio.Play(sound.Click)
	case ebiten.KeyEnter, ebiten.KeyNumpadEnter:
		g.activatePauseAction(buttons[g.pauseSelected].Action)
	case ebiten.KeyM:
		g.activatePauseAction("music")
	case ebiten.KeyV:
		g.activatePauseAction("effects")
	}
}

func (g *Game) activatePauseAction(action string) {
	switch action {
	case "resume":
		g.state = StatePlaying
	case "music", "effects":
		g.audio.Adjust(action == "music")
		g.audio.Play(sound.Click)
	case "quit":
		g.quitSelected = 1 // Cancel is selected until leaving is explicitly confirmed.
		g.state = StateQuitDialog
	}
}
