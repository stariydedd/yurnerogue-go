package game

import (
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/stariydedd/yurnerogue-go/internal/render"
)

func menuPage(state State) (render.MenuPage, bool) {
	switch state {
	case StateMainMenu:
		return render.MenuHome, true
	case StateNameEntry:
		return render.MenuName, true
	case StateHelp, StateLeaderboard, StateStarting, StateDeath, StateWin:
		return render.MenuBack, true
	}
	return 0, false
}

func (g *Game) handleMenuPointer(x, y int) {
	page, ok := menuPage(g.state)
	if !ok {
		return
	}
	action := render.MenuActionAt(g.renderer.Layout, page, x, y)
	if action == "" {
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
