package game

import (
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/stariydedd/yurnerogue-go/internal/render"
	"testing"
)

func TestReferenceScreensHaveOneScrollablePage(t *testing.T) {
	for _, l := range []render.Layout{render.DesktopLayout(), render.TouchLayout(390, 600)} {
		g := New(&render.Renderer{Layout: l})
		for _, state := range []State{StateHelp, StateLeaderboard} {
			g.state, g.helpReturn = state, StatePlaying
			g.leaderboard = make([]render.LeaderboardRecord, 11)
			page, _ := menuPage(state)
			buttons := render.MenuButtons(l, page)
			if len(buttons) != 1 || buttons[0].Action != "back" {
				t.Fatal("tabs or pagination remain")
			}
			if state == StateLeaderboard {
				if _, _, _, ok := g.referenceScrollState(); ok {
					t.Fatal("leaderboard is still scrollable")
				}
				g.HandleKey(ebiten.KeyDown)
				g.HandleKey(ebiten.KeyEnd)
				if g.state != StateLeaderboard || g.beginReferenceDrag(1, 100, 200) {
					t.Fatal("leaderboard input changed screen or captured drag")
				}
				g.topResults = make(chan topResult, 1)
				tapMenuAction(t, g, "back")
				if g.state != StateMainMenu || g.topResults != nil {
					t.Fatal("BACK kept request")
				}
				continue
			}
			view, offset, limit, _ := g.referenceScrollState()
			g.HandleKey(ebiten.KeyEnd)
			if *offset != limit || g.state != state {
				t.Fatal("End did not reach bottom")
			}
			g.HandleKey(ebiten.KeyDown)
			if *offset != limit {
				t.Fatal("scroll exceeded content")
			}
			g.HandleKey(ebiten.KeyHome)
			if *offset != 0 {
				t.Fatal("Home did not reach top")
			}
			g.HandleKey(ebiten.KeyUp)
			if *offset != 0 {
				t.Fatal("negative scroll")
			}
			g.HandleKey(ebiten.KeyPageDown)
			if limit > 0 && *offset == 0 {
				t.Fatal("page scroll failed")
			}
			if g.beginReferenceDrag(1, buttons[0].Bounds.Min.X+10, buttons[0].Bounds.Min.Y+10) {
				t.Fatal("BACK captured by scrolling")
			}
			if limit > 0 && !g.beginReferenceDrag(1, view.Min.X+20, view.Min.Y+20) {
				t.Fatal("content cannot be swiped")
			}
			g.scrollReference(-100000)
			if *offset != 0 {
				t.Fatal("swipe did not clamp")
			}
			g.topResults = make(chan topResult, 1)
			tapMenuAction(t, g, "back")
			if state == StateHelp && g.state != StatePlaying {
				t.Fatal("help lost return screen")
			}
			if state == StateLeaderboard && (g.state != StateMainMenu || g.topResults != nil) {
				t.Fatal("BACK kept request")
			}
		}
	}
}
