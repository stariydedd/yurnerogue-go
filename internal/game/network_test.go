package game

import (
	"errors"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/stariydedd/yurnerogue-go/internal/domain"
	"github.com/stariydedd/yurnerogue-go/internal/leaderboard"
	"github.com/stariydedd/yurnerogue-go/internal/render"
)

func TestNetworkResultsAreAppliedByGameLoop(t *testing.T) {
	g := &Game{topResults: make(chan topResult, 1), submitResults: make(chan submitResult, 1), leaderboardLoading: true}
	g.topResults <- topResult{runs: []leaderboard.Run{{PlayerName: "tester", Treasures: 42}}}
	g.submitResults <- submitResult{}
	g.pollNetwork()
	if g.leaderboardLoading || g.leaderboardSource != "GLOBAL" || len(g.leaderboard) != 1 || g.leaderboard[0].Treasures != 42 {
		t.Fatal("game loop did not apply leaderboard response")
	}
	if g.submitStatus != "Score submitted to global leaderboard!" || g.topResults != nil || g.submitResults != nil {
		t.Fatal("game loop did not consume submission response")
	}
}

func TestLeavingScreenDiscardsLateNetworkResults(t *testing.T) {
	oldTop, oldSubmit := make(chan topResult, 1), make(chan submitResult, 1)
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
	oldSubmit <- submitResult{err: errors.New("old run")}
	g.pollNetwork()
	if !g.leaderboardLoading || g.leaderboardSource != "" || g.submitStatus != "" {
		t.Fatal("late response changed the new screen or run")
	}
}

func TestNetworkFailureDoesNotClaimScoreWasLost(t *testing.T) {
	g := &Game{submitResults: make(chan submitResult, 1)}
	g.submitResults <- submitResult{err: leaderboard.ErrTimeout}
	g.pollNetwork()
	if g.submitStatus != "Score submission could not be confirmed." {
		t.Fatal(g.submitStatus)
	}
}

func TestSubmissionRejectionMessages(t *testing.T) {
	for _, tc := range []struct {
		err  error
		want string
	}{
		{leaderboard.ErrRateLimited, "Too many scores. Submission rejected."},
		{leaderboard.ErrRejected, "Score rejected by the server."},
	} {
		g := &Game{submitResults: make(chan submitResult, 1)}
		g.submitResults <- submitResult{err: tc.err}
		g.pollNetwork()
		if g.submitStatus != tc.want {
			t.Fatal(g.submitStatus)
		}
	}
}

func TestSubmittedRunShowsItsPlace(t *testing.T) {
	gold := func(v int) *int { return &v }
	for _, tc := range []struct {
		name      string
		placement leaderboard.Placement
		want      render.Placement
	}{
		{"podium", leaderboard.Placement{Place: 2, Top10Gold: gold(100)}, render.Placement{Place: 2}},
		{"tenth", leaderboard.Placement{Place: 10, Top10Gold: gold(300)}, render.Placement{Place: 10}},
		{"outside", leaderboard.Placement{Place: 11, Top10Gold: gold(420)}, render.Placement{Place: 11, GoldShort: 121}},
		{"tied with 10th", leaderboard.Placement{Place: 11, Top10Gold: gold(300)}, render.Placement{Place: 11, GoldShort: 1}},
		{"unknown", leaderboard.Placement{}, render.Placement{}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			session := domain.NewSessionSeed(1)
			session.Player.Treasures = 300
			g := &Game{state: StateDeath, session: session, submitResults: make(chan submitResult, 1)}
			g.submitResults <- submitResult{placement: tc.placement}
			g.pollNetwork()
			if g.placement != tc.want {
				t.Fatalf("placement = %+v, want %+v", g.placement, tc.want)
			}
		})
	}
}

func TestFailedSubmissionShowsNoPlace(t *testing.T) {
	g := &Game{state: StateDeath, session: domain.NewSessionSeed(1), submitResults: make(chan submitResult, 1)}
	g.submitResults <- submitResult{placement: leaderboard.Placement{Place: 3}, err: leaderboard.ErrRejected}
	g.pollNetwork()
	if g.placement != (render.Placement{}) {
		t.Fatalf("a rejected run got a place: %+v", g.placement)
	}
}

func TestNewRunForgetsThePlace(t *testing.T) {
	g := &Game{placement: render.Placement{Place: 4}}
	g.startNewGame()
	if g.placement != (render.Placement{}) {
		t.Fatal("the next run inherited the last place")
	}
	g.placement = render.Placement{Place: 4}
	g.returnToMenu()
	if g.placement != (render.Placement{}) {
		t.Fatal("the menu kept the last place")
	}
}
