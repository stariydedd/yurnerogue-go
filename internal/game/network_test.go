package game

import (
	"errors"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/stariydedd/yurnerogue-go/internal/leaderboard"
)

func TestNetworkResultsAreAppliedByGameLoop(t *testing.T) {
	g := &Game{topResults: make(chan topResult, 1), submitResults: make(chan error, 1), leaderboardLoading: true}
	g.topResults <- topResult{runs: []leaderboard.Run{{PlayerName: "tester", Treasures: 42}}}
	g.submitResults <- nil
	g.pollNetwork()
	if g.leaderboardLoading || g.leaderboardSource != "GLOBAL" || len(g.leaderboard) != 1 || g.leaderboard[0].Treasures != 42 {
		t.Fatal("game loop did not apply leaderboard response")
	}
	if g.submitStatus != "Score submitted to global leaderboard!" || g.topResults != nil || g.submitResults != nil {
		t.Fatal("game loop did not consume submission response")
	}
}

func TestLeavingScreenDiscardsLateNetworkResults(t *testing.T) {
	oldTop, oldSubmit := make(chan topResult, 1), make(chan error, 1)
	g := &Game{state: StateLeaderboard, topResults: oldTop, submitResults: oldSubmit}
	g.HandleKey(ebiten.KeyEnter)
	if g.topResults != nil {
		t.Fatal("leaving leaderboard must detach its request")
	}
	g.returnToMenu()
	g.startNewGame()
	g.topResults = make(chan topResult, 1)
	g.leaderboardLoading = true
	oldTop <- topResult{err: errors.New("old request")}
	oldSubmit <- errors.New("old run")
	g.pollNetwork()
	if !g.leaderboardLoading || g.leaderboardSource != "" || g.submitStatus != "" {
		t.Fatal("late response changed the new screen or run")
	}
}

func TestNetworkFailureDoesNotClaimScoreWasLost(t *testing.T) {
	g := &Game{submitResults: make(chan error, 1)}
	g.submitResults <- leaderboard.ErrTimeout
	g.pollNetwork()
	if g.submitStatus != "Score submission could not be confirmed." {
		t.Fatal(g.submitStatus)
	}
}
