package leaderboard

import (
	"regexp"
	"testing"
)

func TestSubmissionIDsAreDistinctUUIDs(t *testing.T) {
	pattern := regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$`)
	seen := make(map[string]bool)
	for range 100 {
		id, err := NewSubmissionID()
		if err != nil || !pattern.MatchString(id) || seen[id] {
			t.Fatalf("invalid or duplicate ID %q: %v", id, err)
		}
		seen[id] = true
	}
}

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
