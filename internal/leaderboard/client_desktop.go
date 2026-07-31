//go:build !js

package leaderboard

import (
	"bytes"
	"io"
	"net/http"
	"os"
)

// New создаёт клиент для нативной сборки. Адрес берётся из ROGUE_API,
// по умолчанию — рабочий сервер игры.
func New() *Client {
	base := os.Getenv("ROGUE_API")
	if base == "" {
		base = "https://yurnerogue.ru"
	}
	return &Client{BaseURL: base}
}

var httpClient = &http.Client{Timeout: Timeout}

// do выполняет обычный HTTP-запрос.
func (c *Client) do(method, path string, body []byte) ([]byte, error) {
	var reader io.Reader
	if body != nil {
		reader = bytes.NewReader(body)
	}
	req, err := http.NewRequest(method, c.BaseURL+path, reader)
	if err != nil {
		return nil, err
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		return nil, ErrUnavailable
	}
	return io.ReadAll(resp.Body)
}
