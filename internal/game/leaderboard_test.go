//go:build !js

package game

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stariydedd/yurnerogue-go/internal/domain"
	"github.com/stariydedd/yurnerogue-go/internal/leaderboard"
)

func TestSubmitRunReportsCompletedLevel(t *testing.T) {
	for _, tc := range []struct {
		name  string
		level int
		win   bool
	}{
		{name: "death on first floor", level: 1},
		{name: "death on last floor", level: domain.MaxLevels},
		{name: "victory", level: domain.MaxLevels, win: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			received := make(chan leaderboard.Run, 1)
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodPost || r.URL.Path != "/api/runs" {
					t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
				}
				var run leaderboard.Run
				if err := json.NewDecoder(r.Body).Decode(&run); err != nil {
					t.Errorf("decode submitted run: %v", err)
				}
				received <- run
				w.WriteHeader(http.StatusCreated)
			}))
			defer server.Close()
			t.Setenv("ROGUE_API", server.URL)

			s := domain.NewSessionAtLevel(tc.level)
			s.Player.Treasures = 1234
			s.Stats.EnemiesKilled = 7
			if tc.win {
				s.Player.X, s.Player.Y = s.Level.Exit.X, s.Level.Exit.Y
				if !s.CheckExit() || !s.Won() {
					t.Fatal("leaving the last floor must win the run")
				}
			} else {
				s.Player.TakeDamage(s.Player.Health)
			}
			g := &Game{session: s, playerName: "tester"}
			g.checkGameOver()

			select {
			case run := <-received:
				if run.Level != tc.level {
					t.Fatalf("submitted level %d, want %d", run.Level, tc.level)
				}
				if run.PlayerName != "tester" || run.Treasures != 1234 || run.EnemiesKilled != 7 {
					t.Fatalf("run details changed: %+v", run)
				}
			case <-time.After(2 * leaderboard.Timeout):
				t.Fatal("run was not submitted")
			}
		})
	}
}
