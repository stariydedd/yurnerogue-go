package game

import (
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/stariydedd/yurnerogue-go/internal/domain"
	"github.com/stariydedd/yurnerogue-go/internal/render"
)

// Draw рисует кадр: сперва в логическую поверхность, затем растягивает её
// на всё окно.
func (g *Game) Draw(screen *ebiten.Image) {
	r := g.renderer
	surface := g.ensureSurface()

	g.drawScreen(surface)

	page, menu := menuPage(g.state)
	if menu {
		r.DrawMenuButtons(surface, page, g.menuSelected)
	} else if g.controls != nil {
		var player *domain.Person
		if g.session != nil {
			player = g.session.Player
		}
		g.controls.Draw(surface, r, g.pressedControls(), selectLabel(g.state), runControlVisible(g.state), player)
	}

	sx, sy := g.scaleToWindow()
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Scale(sx, sy)
	op.Filter = ebiten.FilterNearest // пиксель-арт остаётся чётким
	screen.DrawImage(surface, op)
}

// pressedControls — контролы, которые сейчас удерживаются (для подсветки).
func (g *Game) pressedControls() map[string]bool {
	held := map[string]bool{}
	if g.state == StatePlaying && g.pendingRun {
		held[render.CtrlRun] = true
	}
	if g.touch == nil {
		return held
	}
	for _, control := range g.touch.pressed {
		held[control] = true
	}
	return held
}

// drawScreen рисует содержимое текущего экрана.
func (g *Game) drawScreen(screen *ebiten.Image) {
	r := g.renderer

	switch g.state {
	case StateStarting:
		hint := "Q / Esc / MENU: cancel"
		if r.Layout.Touch {
			hint = ""
		}
		r.DrawEndScreen(screen, "CONNECTING", "Starting game...", hint)
	case StateMainMenu:
		r.DrawMainMenu(screen, g.menuSelected, g.menuMessage)
	case StateNameEntry:
		r.DrawNameEntry(screen, g.nameInput, g.submitStatus)
	case StatePlaying:
		r.DrawWorld(screen, g.session)
		r.DrawHUD(screen, g.session)
	case StateItemMenu:
		r.DrawWorld(screen, g.session)
		r.DrawHUD(screen, g.session)
		r.DrawItemMenu(screen, g.itemMenuItems, g.itemMenuBareHand, g.itemMenuSelected)
	case StateQuitDialog:
		r.DrawWorld(screen, g.session)
		r.DrawHUD(screen, g.session)
		r.DrawQuitDialog(screen, g.quitSelected)
	case StateLeaderboard:
		r.DrawLeaderboard(screen, g.leaderboard, g.leaderboardLoading, g.leaderboardSource)
	case StateHelp:
		r.DrawHelp(screen)
	case StateDeath:
		r.DrawEndScreen(screen, "YOU DIED", g.submitStatus)
	case StateWin:
		r.DrawEndScreen(screen, "YOU WIN!", g.submitStatus)
	}
}
