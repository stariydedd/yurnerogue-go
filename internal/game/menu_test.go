package game

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/stariydedd/yurnerogue-go/internal/render"
)

func TestMobileMenuLeaderboardAndStartUseExistingNetworkFlow(t *testing.T) {
	requests := make(chan string, 2)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests <- r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		if r.Method == http.MethodGet {
			_, _ = w.Write([]byte("[]"))
		} else {
			_, _ = w.Write([]byte(`{"ticket":"test-ticket","seed":"21","version":"1"}`))
		}
	}))
	defer server.Close()
	t.Setenv("ROGUE_API", server.URL)
	g := New(&render.Renderer{Layout: render.TouchLayout(390, 700)})
	tapMenuAction(t, g, "scoreboard")
	if g.state != StateLeaderboard || g.topResults == nil {
		t.Fatal("LEADERBOARD did not start loading")
	}
	select {
	case <-g.topResults:
	case <-time.After(2 * time.Second):
		t.Fatal("leaderboard request timed out")
	}
	tapMenuAction(t, g, "back")
	tapMenuAction(t, g, "new")
	tapMenuAction(t, g, "start")
	if g.state != StateStarting || g.startResults == nil {
		t.Fatal("PLAY did not request a game")
	}
	select {
	case <-g.startResults:
	case <-time.After(2 * time.Second):
		t.Fatal("start request timed out")
	}
	if len(requests) != 2 {
		t.Fatal("menu triggered duplicate requests")
	}
	tapMenuAction(t, g, "back")
	if g.state != StateMainMenu || g.startResults != nil {
		t.Fatal("BACK did not cancel game startup")
	}
}

func tapMenuAction(t *testing.T, g *Game, action string) {
	t.Helper()
	page, ok := menuPage(g.state)
	if !ok {
		t.Fatal("not a menu page")
	}
	for _, button := range render.MenuButtons(g.renderer.Layout, page) {
		if button.Action == action {
			g.handleMenuPointer(button.Bounds.Min.X+button.Bounds.Dx()/2, button.Bounds.Min.Y+button.Bounds.Dy()/2)
			return
		}
	}
	t.Fatalf("missing action %s", action)
}

func TestMobileMenuDirectPlayAndBack(t *testing.T) {
	g := New(&render.Renderer{Layout: render.TouchLayout(390, 700)})
	g.playerName = "tester"
	g.menuSelected = 2
	g.menuMessage = "Previous game ended"
	tapMenuAction(t, g, "new")
	if g.state != StateNameEntry || g.nameInput != "tester" {
		t.Fatal("PLAY did not open name entry")
	}
	tapMenuAction(t, g, "back")
	if g.state != StateMainMenu {
		t.Fatal("BACK did not return to menu")
	}
	g.HandleKey(ebiten.KeyDown)
	if g.menuSelected != 1 {
		t.Fatal("keyboard navigation changed")
	}
}

func TestDesktopMenuPointerAndKeyboardUseSameActions(t *testing.T) {
	l := render.DesktopLayout()
	g := New(&render.Renderer{Layout: l})
	g.HandleKey(ebiten.KeyDown)
	g.HandleKey(ebiten.KeyDown)
	if g.menuSelected != 2 {
		t.Fatal("keyboard selection did not reach HELP")
	}
	g.HandleKey(ebiten.KeyEnter)
	if g.state != StateHelp {
		t.Fatal("keyboard did not open HELP")
	}
	tapMenuAction(t, g, "back")
	if g.state != StateMainMenu {
		t.Fatal("desktop BACK did not return")
	}
	tapMenuAction(t, g, "new")
	if g.state != StateNameEntry {
		t.Fatal("desktop PLAY used stale selection")
	}
	tapMenuAction(t, g, "back")
	tapMenuAction(t, g, "help")
	if g.state != StateHelp {
		t.Fatal("desktop pointer did not open HELP")
	}
}

func TestMobileHelpBackPreservesReturnScreen(t *testing.T) {
	g := New(&render.Renderer{Layout: render.TouchLayout(390, 700)})
	tapMenuAction(t, g, "help")
	if g.state != StateHelp {
		t.Fatal("HELP did not open")
	}
	g.handleMenuPointer(10, 10)
	if g.state != StateHelp {
		t.Fatal("background tap closed help")
	}
	tapMenuAction(t, g, "back")
	if g.state != StateMainMenu {
		t.Fatal("help did not return to menu")
	}
	g.state, g.helpReturn = StateHelp, StatePlaying
	tapMenuAction(t, g, "back")
	if g.state != StatePlaying {
		t.Fatal("help did not return to game")
	}
}

func TestMenuBackDetachesLeaderboardRequest(t *testing.T) {
	g := New(&render.Renderer{Layout: render.TouchLayout(390, 700)})
	g.state = StateLeaderboard
	g.topResults = make(chan topResult, 1)
	tapMenuAction(t, g, "back")
	if g.state != StateMainMenu || g.topResults != nil {
		t.Fatal("BACK left leaderboard request active")
	}
}

func TestMobileMenusDisableGameplayTargets(t *testing.T) {
	l := render.TouchLayout(390, 700)
	ti := newTouchInput(render.NewControls(l))
	for _, state := range []State{StateMainMenu, StateNameEntry, StateStarting, StateHelp, StateLeaderboard, StateWin, StateDeath} {
		if _, ok := menuPage(state); !ok {
			t.Fatalf("gameplay panel visible on %v", state)
		}
		if got := ti.press(1, 104, l.ControlsTop()+54, state, nil); len(got) != 0 {
			t.Fatal("hidden d-pad fired")
		}
	}
	for _, state := range []State{StatePlaying, StateItemMenu, StateQuitDialog} {
		if _, ok := menuPage(state); ok {
			t.Fatal("gameplay controls hidden during game")
		}
	}
}
