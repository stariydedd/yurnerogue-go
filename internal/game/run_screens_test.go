package game

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/stariydedd/yurnerogue-go/internal/domain"
	"github.com/stariydedd/yurnerogue-go/internal/leaderboard"
	"github.com/stariydedd/yurnerogue-go/internal/render"
)

func TestWelcomePrecedesNameAndNetwork(t *testing.T) {
	for _, l := range []render.Layout{render.DesktopLayout(), render.TouchLayout(390, 600)} {
		g := New(&render.Renderer{Layout: l})
		g.playerName = "tester"
		g.HandleKey(ebiten.KeyEnter)
		if g.state != StateWelcome || g.startResults != nil || g.session != nil || g.runTicket != "" {
			t.Fatal("welcome started a run before reading instructions")
		}
		g.HandleKey(ebiten.KeyDown)
		g.HandleKey(ebiten.KeyEnter)
		if g.state != StateMainMenu {
			t.Fatal("welcome MAIN MENU selection ignored")
		}
		g.HandleKey(ebiten.KeyEnter)
		g.HandleKey(ebiten.KeyEnter)
		if g.state != StateNameEntry || g.nameInput != "tester" || g.startResults != nil {
			t.Fatal("continue skipped name entry or lost remembered name")
		}
	}
}

func TestResultsReplayAndBackDetachOldResponses(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var payload map[string]string
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Error(err)
		}
		if r.URL.Path != "/api/runs/start" || payload["player_name"] != "tester" || payload["version"] != domain.RulesVersion {
			t.Error("replay did not request a fresh ticket with the remembered name")
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(leaderboard.Ticket{Ticket: "new-ticket", Seed: "21", Version: domain.RulesVersion})
	}))
	defer server.Close()
	t.Setenv("ROGUE_API", server.URL)
	for _, l := range []render.Layout{render.DesktopLayout(), render.TouchLayout(390, 600)} {
		for _, end := range []State{StateDeath, StateWin} {
			for _, action := range []string{"again", "back"} {
				g := New(&render.Renderer{Layout: l})
				g.startNewGame()
				g.playerName = "tester"
				g.finishRun(end)
				oldSession := g.session
				oldResults := make(chan error, 1)
				g.submitResults = oldResults
				tapMenuAction(t, g, action)
				want := StateMainMenu
				if action == "again" {
					want = StateStarting
				}
				if g.state != want || g.submitResults != nil || g.runTicket != "" || g.playerName != "tester" {
					t.Fatal("results navigation kept old run or lost name")
				}
				if action == "again" {
					if g.session != oldSession || g.startResults == nil {
						t.Fatal("replay did not start loading or lost fallback summary")
					}
					select {
					case result := <-g.startResults:
						g.startResults <- result
						g.pollNetwork()
					case <-time.After(2 * time.Second):
						t.Fatal("replay start timed out")
					}
					want = StatePlaying
					if g.state != want || g.session == oldSession || g.runTicket != "new-ticket" || g.session.Actions() != "" || g.session.Stats != (domain.Stats{}) {
						t.Fatal("replay did not create a fresh run")
					}
				} else if g.session != nil || g.startResults != nil {
					t.Fatal("MAIN MENU kept the old run")
				}
				g.submitStatus = "new screen"
				oldResults <- nil
				g.pollNetwork()
				if g.state != want || g.submitStatus != "new screen" {
					t.Fatal("late submission changed the new screen")
				}
			}
		}
	}
}

func TestReplayStartFailureReturnsToSummaryWithoutNameEntry(t *testing.T) {
	for _, end := range []State{StateDeath, StateWin} {
		for _, result := range []startResult{
			{err: leaderboard.ErrTimeout}, {err: leaderboard.ErrUnavailable},
			{err: leaderboard.ErrRateLimited}, {err: leaderboard.ErrRejected},
			{ticket: leaderboard.Ticket{Ticket: "ticket", Seed: "invalid", Version: domain.RulesVersion}},
		} {
			g := New(&render.Renderer{Layout: render.DesktopLayout()})
			g.startNewGame()
			session := g.session
			g.playerName = "tester"
			g.state, g.startReturn = StateStarting, end
			g.startResults = make(chan startResult, 1)
			g.startResults <- result
			g.pollNetwork()
			if g.state != end || g.session != session || g.playerName != "tester" || g.runTicket != "" || g.submitStatus == "" {
				t.Fatal("failed replay showed name entry or lost summary")
			}
		}
	}
}

func TestResultsKeepStatsWhileSubmissionFinishes(t *testing.T) {
	g := New(&render.Renderer{Layout: render.DesktopLayout()})
	g.startNewGame()
	g.session.Stats.EnemiesKilled = 12
	session := g.session
	g.finishRun(StateDeath)
	g.submitResults = make(chan error, 1)
	g.submitResults <- nil
	g.pollNetwork()
	if g.state != StateDeath || g.session != session || g.session.Stats.EnemiesKilled != 12 || g.submitStatus != "Score submitted to global leaderboard!" {
		t.Fatal("submission erased the run summary")
	}
	g.runMenuSelected = 1
	g.finishRun(StateDeath)
	if g.submitResults != nil || g.runMenuSelected != 1 || g.submitStatus != "Score submitted to global leaderboard!" {
		t.Fatal("finishing twice restarted submission or reset selection")
	}
	g.HandleKey(ebiten.KeyEnter)
	if g.state != StateMainMenu {
		t.Fatal("results keyboard selection ignored")
	}
	g.state = StateWin
	g.HandleKey(ebiten.KeyEscape)
	if g.state != StateMainMenu {
		t.Fatal("Escape did not leave results")
	}
}
