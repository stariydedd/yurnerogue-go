// verifier replays one bounded request in an isolated process. It has no
// database, HTTP listener or client-supplied code, only the shared game rules.
package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"

	"github.com/stariydedd/yurnerogue-go/internal/domain"
)

func main() {
	if len(os.Args) == 2 && os.Args[1] == "--version" {
		fmt.Println(domain.RulesVersion)
		return
	}
	var request struct {
		Seed    int64  `json:"seed"`
		Actions string `json:"actions"`
	}
	decoder := json.NewDecoder(io.LimitReader(os.Stdin, 65537))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&request); err != nil {
		fail()
	}
	var extra any
	if decoder.Decode(&extra) != io.EOF {
		fail()
	}
	s, err := domain.Replay(request.Seed, request.Actions)
	if err != nil {
		fail()
	}
	json.NewEncoder(os.Stdout).Encode(map[string]int{
		"treasures": s.Player.Treasures, "level": min(s.LevelNum, domain.MaxLevels),
		"enemies_killed": s.Stats.EnemiesKilled, "food_used": s.Stats.FoodUsed,
		"elixirs_used": s.Stats.ElixirsUsed, "scrolls_read": s.Stats.ScrollsRead,
		"attacks_made": s.Stats.AttacksMade, "hits_taken": s.Stats.HitsTaken,
		"tiles_moved": s.Stats.TilesMoved,
	})
}

func fail() {
	fmt.Fprintln(os.Stderr, "invalid or unfinished replay")
	os.Exit(2)
}
