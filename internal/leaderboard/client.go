// Package leaderboard — HTTP-клиент глобальной таблицы рекордов.
//
// Бэкенд остался прежним (FastAPI + PostgreSQL), поэтому формат запросов
// совпадает с Python-версией: POST /api/runs и GET /api/leaderboard.
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

// Submit отправляет результат забега.
func (c *Client) Submit(run Run) error {
	body, err := json.Marshal(run)
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
