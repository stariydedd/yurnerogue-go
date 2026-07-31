package game

import (
	"github.com/hajimehoshi/ebiten/v2"
)

// Draw рисует текущий экран и, на тач-раскладке, панель кнопок.
func (g *Game) Draw(screen *ebiten.Image) {
	r := g.renderer

	g.drawScreen(screen)

	// Справке отдаётся весь экран: панель не рисуется, выход — тап.
	if g.controls != nil && g.state != StateHelp {
		g.controls.Draw(screen, r, g.pressedControls(), selectLabel(g.state))
	}
}

// pressedControls — контролы, которые сейчас удерживаются (для подсветки).
func (g *Game) pressedControls() map[string]bool {
	held := map[string]bool{}
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
	case StateMainMenu:
		r.DrawMainMenu(screen, g.menuSelected, g.menuMessage)
	case StateNameEntry:
		r.DrawNameEntry(screen, g.nameInput)
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
