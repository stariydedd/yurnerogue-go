package game

import (
	"errors"
	"strconv"

	"github.com/stariydedd/yurnerogue-go/internal/domain"
	"github.com/stariydedd/yurnerogue-go/internal/leaderboard"
	"github.com/stariydedd/yurnerogue-go/internal/render"
)

// openLeaderboard показывает экран рекордов и тянет данные с сервера.
//
// Каждый запрос получает собственный буфер: старый ответ не меняет новый экран.
// Только pollNetwork в игровом потоке обновляет поля, которые читает Draw.
func (g *Game) openLeaderboard() {
	g.state = StateLeaderboard
	g.leaderboard = nil
	g.leaderboardLoading = true
	g.leaderboardSource = ""

	client := leaderboard.New()
	results := make(chan topResult, 1)
	g.topResults = results
	go func() {
		runs, err := client.Top()
		results <- topResult{runs: runs, err: err}
	}()
}

type topResult struct {
	runs []leaderboard.Run
	err  error
}

func (g *Game) pollNetwork() {
	select {
	case result := <-g.startResults:
		g.startResults = nil
		seed, err := strconv.ParseInt(result.ticket.Seed, 10, 64)
		if result.err != nil || err != nil || result.ticket.Version != domain.RulesVersion || result.ticket.Ticket == "" {
			// Never silently turn a failed server start into an unsubmitable run.
			g.runTicket = ""
			g.session = nil
			g.state = StateNameEntry
			g.submitStatus = startFailureStatus(result.err)
		} else {
			g.startNewGame()
			g.session = domain.NewSessionSeed(seed)
			g.runTicket = result.ticket.Ticket
		}
	default:
	}
	select {
	case result := <-g.topResults:
		g.topResults = nil
		g.leaderboardLoading = false
		if result.err != nil {
			g.leaderboardSource = "SERVER UNAVAILABLE"
			break
		}
		runs := result.runs
		records := make([]render.LeaderboardRecord, 0, len(runs))
		for _, r := range runs {
			records = append(records, render.LeaderboardRecord{
				PlayerName:    r.PlayerName,
				Treasures:     r.Treasures,
				Level:         r.Level,
				EnemiesKilled: r.EnemiesKilled,
				FoodUsed:      r.FoodUsed,
				ElixirsUsed:   r.ElixirsUsed,
				ScrollsRead:   r.ScrollsRead,
				AttacksMade:   r.AttacksMade,
				HitsTaken:     r.HitsTaken,
				TilesMoved:    r.TilesMoved,
			})
		}
		g.leaderboard = records
		g.leaderboardSource = "GLOBAL"
	default:
	}
	select {
	case err := <-g.submitResults:
		g.submitResults = nil
		switch {
		case errors.Is(err, leaderboard.ErrRateLimited):
			g.submitStatus = "Too many scores. Submission rejected."
		case errors.Is(err, leaderboard.ErrRejected):
			g.submitStatus = "Score rejected by the server."
		case err != nil:
			// При потере ответа сервер уже мог сохранить результат.
			g.submitStatus = "Score submission could not be confirmed."
		default:
			g.submitStatus = "Score submitted to global leaderboard!"
		}
	default:
	}
}

func startFailureStatus(err error) string {
	switch {
	case errors.Is(err, leaderboard.ErrTimeout):
		return "Connection timed out. Press PLAY to retry."
	case errors.Is(err, leaderboard.ErrRateLimited):
		return "Too many starts. Wait, then press PLAY."
	case errors.Is(err, leaderboard.ErrRejected):
		return "Server rejected the start. Reload and retry."
	case err != nil:
		return "Cannot reach leaderboard. Press PLAY to retry."
	default:
		return "Invalid server response. Reload and retry."
	}
}

type startResult struct {
	ticket leaderboard.Ticket
	err    error
}

func (g *Game) requestRankedGame() {
	g.state = StateStarting
	g.submitStatus = ""
	results := make(chan startResult, 1)
	g.startResults = results
	name := g.playerName
	if name == "" {
		name = "anonymous"
	}
	client := leaderboard.New()
	go func() {
		ticket, err := client.StartRun(name, domain.RulesVersion)
		results <- startResult{ticket, err}
	}()
}

// submitRun sends only the ticket and legal input log, never trusted scores.
func (g *Game) submitRun() {
	s := g.session
	if s == nil || g.submitResults != nil {
		return
	}
	if g.runTicket == "" {
		g.submitStatus = "Score not submitted: leaderboard unavailable for this game."
		return
	}
	if s.ReplayOverflow {
		g.submitStatus = "Game length limit reached: score cannot be submitted."
		return
	}
	ticket, actions := g.runTicket, s.Actions()
	g.submitStatus = "Submitting score..."
	client := leaderboard.New()
	results := make(chan error, 1)
	g.submitResults = results
	go func() { results <- client.SubmitReplay(ticket, actions) }()
}
