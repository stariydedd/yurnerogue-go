package domain

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"
)

// Walk towards the nearest enemy using only legal cardinal actions.
func testAction(s *Session) string {
	if s.Player.Sleeping {
		return "z"
	}
	type node struct {
		p     Point
		first string
	}
	q := []node{{Point{s.Player.X, s.Player.Y}, ""}}
	seen := map[Point]bool{q[0].p: true}
	for len(q) > 0 {
		n := q[0]
		q = q[1:]
		for i, d := range dirs4 {
			p := Point{n.p.X + d.X, n.p.Y + d.Y}
			if seen[p] || !s.CanMoveTo(p.X, p.Y) {
				continue
			}
			seen[p] = true
			first := n.first
			if first == "" {
				first = string("wsad"[i])
			}
			if s.OpponentAt(p.X, p.Y) != nil || p == s.Level.Exit {
				return first
			}
			q = append(q, node{p, first})
		}
	}
	return "w"
}

func TestSeededSessionsReplayIndependently(t *testing.T) {
	for seed := int64(1); seed <= 10; seed++ {
		s, twin := NewSessionSeed(seed), NewSessionSeed(seed)
		for i := 0; i < 3000 && s.Player.IsAlive() && !s.Won(); i++ {
			a := testAction(s)
			if err := s.ApplyAction(a); err != nil {
				t.Fatal(err)
			}
			// Advance an unrelated generator between matching actions.
			if i%100 == 0 {
				NewSession()
			}
			if err := twin.ApplyAction(a); err != nil {
				t.Fatal(err)
			}
			left, _ := json.Marshal(s.Player)
			right, _ := json.Marshal(twin.Player)
			if string(left) != string(right) || !reflect.DeepEqual(s.Stats, twin.Stats) {
				t.Fatalf("sessions diverged: seed %d action %d", seed, i)
			}
		}
		replayed, err := Replay(seed, s.Actions())
		if err != nil {
			t.Fatalf("seed %d: %v", seed, err)
		}
		if !reflect.DeepEqual(s.Stats, replayed.Stats) || s.Player.Treasures != replayed.Player.Treasures {
			t.Fatal("wrong score")
		}
		if seed == 1 {
			t.Logf("fixture actions=%q treasures=%d stats=%+v", s.Actions(), s.Player.Treasures, s.Stats)
		}
	}
}

func TestReplayRejectsInvalidAndUnfinishedRuns(t *testing.T) {
	for _, actions := range []string{"", "teleport", "h", "h0", "h9", "z", "w", strings.Repeat("w", MaxReplayBytes+1)} {
		if _, err := Replay(1, actions); err == nil {
			t.Fatalf("accepted %q", actions)
		}
	}
	s := NewSessionSeed(1)
	for s.Player.IsAlive() && !s.Won() {
		s.ApplyAction(testAction(s))
	}
	if _, err := Replay(1, s.Actions()+"w"); err == nil {
		t.Fatal("accepted action after death")
	}
}

func TestGoldenReplayMatchesServerOnEveryPlatform(t *testing.T) {
	const actions = "sssddsssdddddddddddddddsssssssssssssswwwwaaaaaaaaaaaaaaawwwwwwwwwddddddddddddddddsssdddddddddddddddddsssssdddddddssssddddddddddsssssddddddddddddddddddddssddddddddddddddsdddwsdsssaaaaassswwwwwwwwwwwaaaaaaaawwwdddddwwwwwwwwwawwwaawwwwwwwwwwwwaaasaaaassssdddsssssaa"
	s, err := Replay(1, actions)
	if err != nil {
		t.Fatal(err)
	}
	if s.Player.Treasures != 278 || s.Stats != (Stats{EnemiesKilled: 6, AttacksMade: 45, HitsTaken: 20, TilesMoved: 217}) {
		t.Fatalf("rules version %s changed: gold=%d stats=%+v", RulesVersion, s.Player.Treasures, s.Stats)
	}
}

func TestSeededFloorsAndInventoryAreDeterministic(t *testing.T) {
	a, b := NewSessionSeed(777), NewSessionSeed(777)
	for floor := 1; floor <= MaxLevels; floor++ {
		// Include item rolls derived from stats and weapon replacement/drop.
		for _, s := range []*Session{a, b} {
			s.Player.Backpack = append(s.Player.Backpack, NewFood(s.Player), NewElixir(s.Player), NewScroll(s.Player), NewWeapon(s.Player.rng), NewWeapon(s.Player.rng))
			for _, action := range []string{"j0", "k0", "e0", "h1", "h1", "h0", "W", "D", "S", "A"} {
				if s.Player.IsAlive() {
					s.ApplyAction(action)
				}
			}
		}
		// JSON intentionally excludes RNG internals; compare generated geometry,
		// monsters and items instead of pointer identity.
		snapshot := func(s *Session) string {
			data, err := json.Marshal([]any{s.Player, s.Stats, s.Level.Rooms, s.Level.Passages, s.Level.Items, s.Level.Exit})
			if err != nil {
				t.Fatal(err)
			}
			return string(data)
		}
		if snapshot(a) != snapshot(b) {
			t.Fatalf("floor %d diverged", floor)
		}
		a.UpdateLevel()
		NewSession()
		b.UpdateLevel()
	}
}
