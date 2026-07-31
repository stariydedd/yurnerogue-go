package game

import (
	"github.com/stariydedd/yurnerogue-go/internal/leaderboard"
	"github.com/stariydedd/yurnerogue-go/internal/render"
)

// openLeaderboard показывает экран рекордов и тянет данные с сервера.
//
// Запрос уходит в отдельной горутине: в WASM Ebitengine крутит игровой цикл
// в одном потоке, и синхронный запрос подвесил бы картинку. Результат
// присваивается полям, которые читает Draw в том же цикле.
func (g *Game) openLeaderboard() {
	g.state = StateLeaderboard
	g.leaderboard = nil
	g.leaderboardLoading = true
	g.leaderboardSource = ""

	client := leaderboard.New()
	go func() {
		runs, err := client.Top()
		if err != nil {
			g.leaderboardLoading = false
			g.leaderboardSource = "SERVER UNAVAILABLE"
			return
		}
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
		g.leaderboardLoading = false
	}()
}

// submitRun отправляет результат завершённого забега.
func (g *Game) submitRun() {
	s := g.session
	if s == nil {
		return
	}
	name := g.playerName
	if name == "" {
		name = "anonymous"
	}
	run := leaderboard.Run{
		PlayerName:    name,
		Treasures:     s.Player.Treasures,
		Level:         s.LevelNum,
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
	go func() {
		if err := client.Submit(run); err != nil {
			g.submitStatus = "Server unavailable - score not saved."
			return
		}
		g.submitStatus = "Score submitted to global leaderboard!"
	}()
}
