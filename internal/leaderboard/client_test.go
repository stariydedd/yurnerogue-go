package leaderboard

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestRunMarshalsToBackendSchema(t *testing.T) {
	// Имена полей должны совпадать со схемой FastAPI: бэкенд общий с прежней
	// версией игры и переименований не переживёт.
	run := Run{
		PlayerName:    "stan",
		Treasures:     2484,
		Level:         9,
		EnemiesKilled: 31,
		FoodUsed:      5,
		ElixirsUsed:   2,
		ScrollsRead:   1,
		AttacksMade:   120,
		HitsTaken:     44,
		TilesMoved:    900,
	}

	data, err := json.Marshal(run)
	if err != nil {
		t.Fatal(err)
	}
	body := string(data)

	for _, field := range []string{
		`"player_name":"stan"`, `"treasures":2484`, `"level":9`,
		`"enemies_killed":31`, `"food_used":5`, `"elixirs_used":2`,
		`"scrolls_read":1`, `"attacks_made":120`, `"hits_taken":44`,
		`"tiles_moved":900`,
	} {
		if !strings.Contains(body, field) {
			t.Fatalf("в запросе нет %s: %s", field, body)
		}
	}
}

func TestRunUnmarshalsServerResponse(t *testing.T) {
	response := `[
		{"player_name":"stan","treasures":2484,"level":9,"enemies_killed":31,
		 "food_used":5,"elixirs_used":2,"scrolls_read":1,"attacks_made":120,
		 "hits_taken":44,"tiles_moved":900},
		{"player_name":"anonymous","treasures":700,"level":4,"enemies_killed":10,
		 "food_used":2,"elixirs_used":0,"scrolls_read":0,"attacks_made":50,
		 "hits_taken":20,"tiles_moved":300}
	]`

	var runs []Run
	if err := json.Unmarshal([]byte(response), &runs); err != nil {
		t.Fatal(err)
	}
	if len(runs) != 2 {
		t.Fatalf("разобрано %d записей, ожидалось 2", len(runs))
	}
	if runs[0].PlayerName != "stan" || runs[0].Treasures != 2484 || runs[0].Level != 9 {
		t.Fatalf("первая запись разобрана неверно: %+v", runs[0])
	}
	if runs[1].ElixirsUsed != 0 || runs[1].TilesMoved != 300 {
		t.Fatalf("вторая запись разобрана неверно: %+v", runs[1])
	}
}

func TestUnknownFieldsDoNotBreakParsing(t *testing.T) {
	// Бэкенд может добавить поле (например, дату) — игра не должна падать.
	var runs []Run
	err := json.Unmarshal([]byte(`[{"player_name":"a","treasures":1,"created_at":"2026-07-31"}]`), &runs)
	if err != nil {
		t.Fatalf("лишнее поле сломало разбор: %v", err)
	}
	if len(runs) != 1 || runs[0].PlayerName != "a" {
		t.Fatalf("запись разобрана неверно: %+v", runs)
	}
}
