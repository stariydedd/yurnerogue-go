package game

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/stariydedd/yurnerogue-go/internal/render"
	"github.com/stariydedd/yurnerogue-go/internal/sound"
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
	tapMenuAction(t, g, "continue")
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
	if g.state != StateWelcome {
		t.Fatal("PLAY did not open welcome screen")
	}
	tapMenuAction(t, g, "continue")
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
	if g.state != StateWelcome {
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
	for _, state := range []State{StateMainMenu, StateWelcome, StateNameEntry, StateStarting, StateHelp, StateLeaderboard, StateWin, StateDeath, StatePauseMenu, StateQuitDialog} {
		if _, ok := menuPage(state); !ok {
			t.Fatalf("gameplay panel visible on %v", state)
		}
		if got := ti.press(1, 104, l.ControlsTop()+54, state, nil); len(got) != 0 {
			t.Fatal("hidden d-pad fired")
		}
	}
	for _, state := range []State{StatePlaying, StateItemMenu} {
		if _, ok := menuPage(state); ok {
			t.Fatal("gameplay controls hidden during game")
		}
	}
}

func TestPauseMenuVolumeAndResumePreserveRun(t *testing.T) {
	config := t.TempDir()
	t.Setenv("APPDATA", config)
	t.Setenv("XDG_CONFIG_HOME", config)
	for _, l := range []render.Layout{render.DesktopLayout(), render.TouchLayout(390, 600)} {
		g := New(&render.Renderer{Layout: l})
		// No audio device is needed to exercise settings and input routing.
		g.audio = &sound.Engine{}
		g.startNewGame()
		g.runTicket = "ticket-to-preserve"
		g.pendingRun = true
		session, stats, actions := g.session, g.session.Stats, g.session.Actions()
		g.HandleKey(keyForControl(render.CtrlMenu, StatePlaying))
		if g.state != StatePauseMenu || g.pendingRun {
			t.Fatal("MENU did not open pause menu or clear pending RUN")
		}
		for _, volume := range []int{25, 50, 75, 100, 0} {
			g.HandleKey(ebiten.KeyM)
			if got := g.audio.Settings(); got.Music != volume || got.Effects != 0 {
				t.Fatalf("music control changed wrong volume: %+v", got)
			}
		}
		g.HandleKey(ebiten.KeyDown)
		g.HandleKey(ebiten.KeyDown)
		g.HandleKey(ebiten.KeyEnter)
		if got := g.audio.Settings(); got.Effects != 25 || got.Music != 0 {
			t.Fatalf("keyboard volume control failed: %+v", got)
		}
		g.HandleKey(ebiten.KeyM)
		g.HandleKey(ebiten.KeyV)
		if got := g.audio.Settings(); got != (sound.Settings{Music: 25, Effects: 50}) {
			t.Fatalf("volume shortcuts failed: %+v", got)
		}
		tapMenuAction(t, g, "resume")
		if g.state != StatePlaying || g.session != session || g.session.Stats != stats || g.session.Actions() != actions || g.runTicket != "ticket-to-preserve" {
			t.Fatal("pause settings changed or lost run")
		}
		g.HandleKey(ebiten.KeyQ)
		tapMenuAction(t, g, "quit")
		if g.state != StateQuitDialog || g.session != session {
			t.Fatal("exit skipped confirmation")
		}
		g.HandleKey(ebiten.KeyEnter) // Safe default: cancel.
		if g.state != StatePlaying || g.session != session {
			t.Fatal("default confirmation discarded run")
		}
		g.HandleKey(ebiten.KeyQ)
		g.HandleKey(ebiten.KeyEscape)
		if g.state != StatePlaying {
			t.Fatal("Escape did not resume")
		}
		g.HandleKey(ebiten.KeyQ)
		tapMenuAction(t, g, "quit")
		g.HandleKey(ebiten.KeyUp)
		g.HandleKey(ebiten.KeyEnter)
		if g.state != StateMainMenu || g.session != nil || g.runTicket != "" {
			t.Fatal("confirmed exit did not end run")
		}
	}
}

func TestQuitDialogPointerUsesVisibleButtons(t *testing.T) {
	for _, l := range []render.Layout{render.DesktopLayout(), render.TouchLayout(390, 600)} {
		g := New(&render.Renderer{Layout: l})
		g.startNewGame()
		g.runTicket = "keep-until-confirmed"
		session := g.session
		g.HandleKey(ebiten.KeyQ)
		tapMenuAction(t, g, "quit")
		if g.quitSelected != 1 {
			t.Fatal("cancel is not the safe default")
		}
		g.handleMenuPointer(5, 5)
		if g.state != StateQuitDialog {
			t.Fatal("background tap dismissed confirmation")
		}
		tapMenuAction(t, g, "cancel")
		if g.state != StatePlaying || g.session != session || g.runTicket != "keep-until-confirmed" {
			t.Fatal("cancel lost the run")
		}
		g.HandleKey(ebiten.KeyQ)
		tapMenuAction(t, g, "quit")
		tapMenuAction(t, g, "menu")
		if g.state != StateMainMenu || g.session != nil || g.runTicket != "" {
			t.Fatal("confirmed exit did not discard the run")
		}
	}
}
