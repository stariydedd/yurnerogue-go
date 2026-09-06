package leaderboard

import (
	"testing"
)

func TestSubmissionResponseErrors(t *testing.T) {
	for _, tc := range []struct {
		status int
		want   error
	}{
		{429, ErrRateLimited}, {409, ErrRejected}, {422, ErrRejected},
		{413, ErrRejected}, {500, ErrUnavailable}, {503, ErrUnavailable},
	} {
		if got := responseError(tc.status); got != tc.want {
			t.Fatalf("status %d: got %v, want %v", tc.status, got, tc.want)
		}
	}
}
