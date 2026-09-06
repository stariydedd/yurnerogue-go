//go:build !js

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
)

func waitSubmission(t *testing.T, g *Game) {
	t.Helper()
	select {
	case err := <-g.submitResults:
		g.submitResults <- err
		g.pollNetwork()
	case <-time.After(2 * leaderboard.Timeout):
		t.Fatal("submission did not complete")
	}
}

func TestRankedSubmissionContainsOnlyTicketAndReplay(t *testing.T) {
	received := make(chan map[string]string, 3)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var payload map[string]string
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Error(err)
		}
		received <- payload
		w.WriteHeader(201)
	}))
	defer server.Close()
	t.Setenv("ROGUE_API", server.URL)
	g := &Game{session: domain.NewSessionSeed(1), runTicket: "server-issued"}
	g.session.ApplyAction("s")
	g.submitRun()
	pending := g.submitResults
	g.submitRun()
	if g.submitResults != pending {
		t.Fatal("duplicate in-flight request")
	}
	waitSubmission(t, g)
	first := <-received
	if len(first) != 2 || first["ticket"] != "server-issued" || first["actions"] != "s" {
		t.Fatal(first)
	}
	g.submitRun()
	waitSubmission(t, g)
	again := <-received
	if again["ticket"] != first["ticket"] || again["actions"] != first["actions"] {
		t.Fatal("retry changed")
	}
	g.startNewGame()
	if g.runTicket != "" {
		t.Fatal("new practice run kept ticket")
	}
	g.submitRun()
	if g.submitResults != nil || g.submitStatus != "Practice run: not submitted to leaderboard." {
		t.Fatal("practice submitted")
	}
}

func TestLateStartResponseIsDiscarded(t *testing.T) {
	old := make(chan startResult, 1)
	g := &Game{state: StateStarting, startResults: old}
	g.HandleKey(ebiten.KeyQ)
	old <- startResult{ticket: leaderboard.Ticket{Ticket: "late", Seed: "1", Version: domain.RulesVersion}}
	g.pollNetwork()
	if g.state != StateMainMenu || g.session != nil || g.runTicket != "" {
		t.Fatal("cancelled start changed screen")
	}
}

func TestStartResponseCreatesSeededOrPracticeRun(t *testing.T) {
	for _, failure := range []bool{false, true} {
		results := make(chan startResult, 1)
		g := &Game{state: StateStarting, startResults: results}
		result := startResult{ticket: leaderboard.Ticket{Ticket: "ticket", Seed: "1", Version: domain.RulesVersion}}
		if failure {
			result.err = leaderboard.ErrUnavailable
		}
		results <- result
		g.pollNetwork()
		if g.state != StatePlaying || g.session == nil {
			t.Fatal("game did not start")
		}
		if (g.runTicket == "") != failure {
			t.Fatal("wrong ranked state")
		}
		if !failure && g.session.Level.Seed != domain.NewSessionSeed(1).Level.Seed {
			t.Fatal("server seed not used")
		}
	}
}

func TestInventoryInputRecordsSharedDomainAction(t *testing.T) {
	g := &Game{session: domain.NewSessionSeed(1), state: StatePlaying}
	g.session.Player.Backpack = append(g.session.Player.Backpack, domain.NewWeapon())
	g.HandleKey(ebiten.KeyH)
	g.HandleKey(ebiten.Key1)
	if g.session.Actions() != "h1" || g.session.Player.Weapon == nil {
		t.Fatal("inventory bypassed replay path")
	}
}

func TestEndScreenHasNoRetryControls(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("end-screen input unexpectedly submitted a score")
		w.WriteHeader(http.StatusBadRequest)
	}))
	defer server.Close()
	t.Setenv("ROGUE_API", server.URL)
	for _, state := range []State{StateDeath, StateWin} {
		g := &Game{state: state, session: domain.NewSessionSeed(1), runTicket: "ticket", submitStatus: "saved"}
		if runControlVisible(state) || keyForControl("run", state) != ebiten.KeyMax {
			t.Fatal("end screen exposes RUN")
		}
		g.HandleKey(ebiten.KeyR)
		g.HandleKey(keyForControl("run", state))
		if g.submitResults != nil || g.submitStatus != "saved" {
			t.Fatal("retry is still enabled")
		}
	}
	if !runControlVisible(StatePlaying) || keyForControl("run", StatePlaying) != ebiten.KeyF {
		t.Fatal("ordinary gameplay running was removed")
	}
}
