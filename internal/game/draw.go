package game

import (
	"github.com/hajimehoshi/ebiten/v2"
)

// Draw рисует текущий экран.
func (g *Game) Draw(screen *ebiten.Image) {
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
