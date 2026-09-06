//go:build js

package leaderboard

import (
	"errors"
	"syscall/js"
	"testing"
	"time"
)

func TestFetchTimeoutIncludesResponseBody(t *testing.T) {
	for _, bodyStalls := range []bool{false, true} {
		name := "headers"
		if bodyStalls {
			name = "body"
		}
		t.Run(name, func(t *testing.T) {
			original := js.Global().Get("fetch")
			defer js.Global().Set("fetch", original)
			// Настоящий JS Promise отклоняется по abort, как браузерный fetch.
			factory := js.Global().Get("Function").New("bodyStalls", `
				return function(url, opts) {
					const pending = () => new Promise((resolve, reject) => {
						opts.signal.addEventListener('abort', () => reject(new Error('aborted')), {once:true});
					});
					return bodyStalls ? Promise.resolve({ok:true, text:pending}) : pending();
				};`)
			js.Global().Set("fetch", factory.Invoke(bodyStalls))
			_, err := New().doWithTimeout("GET", "/api/leaderboard", nil, 20*time.Millisecond)
			if !errors.Is(err, ErrTimeout) {
				t.Fatalf("expected timeout, got %v", err)
			}
		})
	}
}

func TestFetchSuccessAndRejectedPromise(t *testing.T) {
	for _, tc := range []struct {
		name, script string
		wantErr      bool
	}{
		{"success", `return Promise.resolve({ok:true, text:() => Promise.resolve('[]')});`, false},
		{"string rejection", `return Promise.reject('offline');`, true},
		{"null rejection", `return Promise.reject(null);`, true},
		{"body rejection", `return Promise.resolve({ok:true, text:() => Promise.reject(new Error('lost body'))});`, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			original := js.Global().Get("fetch")
			defer js.Global().Set("fetch", original)
			js.Global().Set("fetch", js.Global().Get("Function").New(tc.script))
			body, err := New().doWithTimeout("GET", "/api/leaderboard", nil, time.Second)
			if (err != nil) != tc.wantErr {
				t.Fatalf("unexpected error: %v", err)
			}
			if !tc.wantErr && string(body) != "[]" {
				t.Fatalf("unexpected body: %s", body)
			}
		})
	}
}

func TestFetchReportsRateLimitAndRejection(t *testing.T) {
	for _, tc := range []struct {
		status int
		want   error
	}{{429, ErrRateLimited}, {409, ErrRejected}, {422, ErrRejected}, {503, ErrUnavailable}} {
		original := js.Global().Get("fetch")
		factory := js.Global().Get("Function").New("status", `return () => Promise.resolve({ok:false, status});`)
		js.Global().Set("fetch", factory.Invoke(tc.status))
		_, err := New().doWithTimeout("POST", "/api/runs", []byte(`{}`), time.Second)
		js.Global().Set("fetch", original)
		if !errors.Is(err, tc.want) {
			t.Fatalf("status %d: got %v, want %v", tc.status, err, tc.want)
		}
	}
}
