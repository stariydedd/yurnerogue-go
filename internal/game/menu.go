package game

import (
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/stariydedd/yurnerogue-go/internal/locale"
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
	case StateQuitDialog:
		return render.MenuQuit, true
	case StateWelcome:
		return render.MenuWelcome, true
	case StateDeath, StateWin:
		return render.MenuResults, true
	case StateHelp:
		return render.MenuHelp, true
	case StateLeaderboard:
		return render.MenuLeaderboard, true
	case StateStarting:
		return render.MenuBack, true
	}
	return 0, false
}

func (g *Game) handleMenuPointer(x, y int) {
	if g.state == StateMainMenu {
		if language := render.LanguageAt(g.renderer.Layout, x, y); language != "" {
			g.renderer.Layout.Language = language
			locale.Save(language)
			return
		}
	}
	if g.setAudioSliderAt(x, y) {
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
	if g.state == StateWelcome || g.state == StateDeath || g.state == StateWin {
		g.activateRunAction(action)
		return
	}
	if g.state == StatePauseMenu {
		g.activatePauseAction(action)
		return
	}
	if g.state == StateQuitDialog {
		for i, option := range render.QuitOptions {
			if option.Key == action {
				g.quitSelected = i
				g.HandleKey(ebiten.KeyEnter)
				return
			}
		}
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
	if action == "start" {
		g.HandleKey(ebiten.KeyEnter)
	} else if action == "back" {
		g.HandleKey(ebiten.KeyEscape)
	}
}

func (g *Game) openWelcome() {
	g.returnToMenu()
	g.submitStatus = ""
	g.runMenuSelected = 0
	g.state = StateWelcome
}

func (g *Game) handleRunMenu(key ebiten.Key) {
	switch key {
	case ebiten.KeyEscape, ebiten.KeyQ:
		g.returnToMenu()
	case ebiten.KeyUp, ebiten.KeyW, ebiten.KeyDown, ebiten.KeyS:
		g.runMenuSelected = 1 - g.runMenuSelected
		g.audio.Play(sound.Click)
	case ebiten.KeyEnter, ebiten.KeyNumpadEnter:
		page, _ := menuPage(g.state)
		g.activateRunAction(render.MenuButtons(g.renderer.Layout, page)[g.runMenuSelected].Action)
	}
}

func (g *Game) activateRunAction(action string) {
	switch action {
	case "continue":
		g.nameInput = g.playerName
		g.state = StateNameEntry
	case "again":
		// Keep the summary for a failed start, but detach the old submission.
		g.submitResults = nil
		g.runTicket = ""
		g.requestRankedGame()
	case "back":
		g.returnToMenu()
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
	case ebiten.KeyLeft, ebiten.KeyA, ebiten.KeyRight, ebiten.KeyD:
		g.adjustAudioSlider(buttons[g.pauseSelected].Action, key)
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
