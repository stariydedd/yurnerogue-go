//go:build !js

package leaderboard

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestSubmitReplayReadsPlacement(t *testing.T) {
	for _, tc := range []struct {
		name, body string
		status     int
		place      int
		gold       int // -1: no top 10 yet
		err        error
	}{
		{"placed", `{"id":7,"treasures":502,"place":12,"top10_gold":900}`, 201, 12, 900, nil},
		{"top 10 not full", `{"id":7,"place":3,"top10_gold":null}`, 201, 3, -1, nil},
		{"old server", `{"id":7,"treasures":502}`, 201, 0, -1, nil},
		{"broken body", `not json`, 200, 0, -1, nil},
		{"rejected", `{"detail":"bad"}`, 409, 0, -1, ErrRejected},
	} {
		t.Run(tc.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != "POST" || r.URL.Path != "/api/runs" {
					t.Errorf("unexpected request %s %s", r.Method, r.URL.Path)
				}
				w.WriteHeader(tc.status)
				w.Write([]byte(tc.body))
			}))
			defer server.Close()
			got, err := (&Client{BaseURL: server.URL}).SubmitReplay("ticket", "d")
			if err != tc.err {
				t.Fatalf("err = %v, want %v", err, tc.err)
			}
			gold := -1
			if got.Top10Gold != nil {
				gold = *got.Top10Gold
			}
			if got.Place != tc.place || gold != tc.gold {
				t.Fatalf("placement = %d/%d, want %d/%d", got.Place, gold, tc.place, tc.gold)
			}
		})
	}
}
