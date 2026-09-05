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
