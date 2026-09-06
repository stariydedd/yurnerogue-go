// Package leaderboard — HTTP-клиент глобальной таблицы рекордов.
//
// FastAPI issues run tickets and verifies input replays before writing scores.
//
// Транспорт разный по платформам: в браузере используется родной fetch, а не
// net/http — тот тянет в WASM TLS и HTTP/2 и стоит около 1.7 МБ в gzip.
package leaderboard

import (
	"encoding/json"
	"errors"
	"time"
)

// Timeout — запрос не должен подвешивать игру: при недоступном сервере
// показывается сообщение, а не бесконечная загрузка.
const Timeout = 5 * time.Second

// Run — результат забега; поля совпадают со схемой бэкенда.
type Run struct {
	Verified      bool   `json:"verified"`
	PlayerName    string `json:"player_name"`
	Treasures     int    `json:"treasures"`
	Level         int    `json:"level"`
	EnemiesKilled int    `json:"enemies_killed"`
	FoodUsed      int    `json:"food_used"`
	ElixirsUsed   int    `json:"elixirs_used"`
	ScrollsRead   int    `json:"scrolls_read"`
	AttacksMade   int    `json:"attacks_made"`
	HitsTaken     int    `json:"hits_taken"`
	TilesMoved    int    `json:"tiles_moved"`
}

// Client общается с API лидерборда.
type Client struct {
	// BaseURL пуст в браузере: путь относительный, значит тот же origin,
	// откуда пришла игра.
	BaseURL string
}

// ErrUnavailable — сервер недоступен или ответил ошибкой.
var ErrUnavailable = errors.New("leaderboard unavailable")

var ErrTimeout = errors.New("leaderboard request timed out")

var ErrRateLimited = errors.New("too many leaderboard submissions")
var ErrRejected = errors.New("leaderboard rejected submission")

func responseError(status int) error {
	switch status {
	case 429:
		return ErrRateLimited
	case 400, 404, 409, 410, 413, 422:
		return ErrRejected
	default:
		return ErrUnavailable
	}
}

type Ticket struct {
	Ticket  string `json:"ticket"`
	Seed    string `json:"seed"`
	Version string `json:"version"`
}

func (c *Client) StartRun(name, version string) (Ticket, error) {
	body, err := json.Marshal(map[string]string{"player_name": name, "version": version})
	if err != nil {
		return Ticket{}, err
	}
	data, err := c.do("POST", "/api/runs/start", body)
	if err != nil {
		return Ticket{}, err
	}
	var ticket Ticket
	err = json.Unmarshal(data, &ticket)
	return ticket, err
}

func (c *Client) SubmitReplay(ticket, actions string) error {
	body, err := json.Marshal(map[string]string{"ticket": ticket, "actions": actions})
	if err != nil {
		return err
	}
	_, err = c.do("POST", "/api/runs", body)
	return err
}

// Top запрашивает таблицу рекордов.
func (c *Client) Top() ([]Run, error) {
	data, err := c.do("GET", "/api/leaderboard", nil)
	if err != nil {
		return nil, err
	}
	var runs []Run
	if err := json.Unmarshal(data, &runs); err != nil {
		return nil, err
	}
	return runs, nil
}
