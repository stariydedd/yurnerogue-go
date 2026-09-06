package game

import (
	"errors"

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

// submitRun отправляет результат завершённого забега.
func (g *Game) submitRun() {
	s := g.session
	if s == nil || g.submitResults != nil {
		return
	}
	if g.submissionID == "" {
		id, err := leaderboard.NewSubmissionID()
		if err != nil {
			g.submitStatus = "Could not prepare score submission."
			return
		}
		g.submissionID = id
	}
	name := g.playerName
	if name == "" {
		name = "anonymous"
	}
	run := leaderboard.Run{
		SubmissionID: g.submissionID,
		PlayerName:   name,
		Treasures:    s.Player.Treasures,
		// После победы сессия уже на уровне 22, но пройденный этаж — 21.
		Level:         min(s.LevelNum, domain.MaxLevels),
		EnemiesKilled: s.Stats.EnemiesKilled,
		FoodUsed:      s.Stats.FoodUsed,
		ElixirsUsed:   s.Stats.ElixirsUsed,
		ScrollsRead:   s.Stats.ScrollsRead,
		AttacksMade:   s.Stats.AttacksMade,
		HitsTaken:     s.Stats.HitsTaken,
		TilesMoved:    s.Stats.TilesMoved,
	}

	g.submitStatus = "Submitting score..."
	client := leaderboard.New()
	results := make(chan error, 1)
	g.submitResults = results
	go func() {
		results <- client.Submit(run)
	}()
}
