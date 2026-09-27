package domain

import (
	"math/rand"
	"testing"
)

// Riki blinks to a random cell of its room; the hero's cell must never be one.
func TestRikiNeverBlinksOntoTheHero(t *testing.T) {
	room := &Room{X: 10, Y: 10, W: 4, H: 1}
	rooms := []*Room{room}
	hero := Point{13, 10} // farther than Riki's hostility radius
	for seed := int64(0); seed < 200; seed++ {
		o := NewOpponent(Ghost)
		o.rng = rand.New(rand.NewSource(seed))
		o.X, o.Y = 10, 10
		room.Enemies = []*Opponent{o}
		for i := 0; i < 20; i++ {
			o.Move(hero.X, hero.Y, Point{-1, -1}, rooms, nil, room.Enemies)
			if (Point{o.X, o.Y}) == hero {
				t.Fatalf("seed %d: Riki blinked onto the hero", seed)
			}
			o.X, o.Y = 10, 10 // keep it out of reach so it keeps blinking
		}
	}
}

// No enemy may ever share the hero's cell, whatever its movement pattern.
func TestEnemiesNeverShareTheHeroCell(t *testing.T) {
	for seed := int64(1); seed <= 40; seed++ {
		s := NewSessionSeed(seed)
		for turn := 0; turn < 1000 && s.Player.IsAlive() && !s.Won(); turn++ {
			if err := s.ApplyAction(testAction(s)); err != nil {
				t.Fatal(err)
			}
			for _, o := range s.Level.AllOpponents() {
				if o.IsAlive() && o.X == s.Player.X && o.Y == s.Player.Y {
					t.Fatalf("seed %d turn %d: %s stands on the hero", seed, turn, o.Type.DisplayName())
				}
			}
		}
	}
}

// Riki's blink used to pick any free cell of the room, the portal included.
func TestRikiNeverBlinksOntoThePortal(t *testing.T) {
	room := &Room{X: 10, Y: 10, W: 3, H: 1}
	rooms := []*Room{room}
	hero, exit := Point{40, 40}, Point{12, 10}
	for seed := int64(0); seed < 200; seed++ {
		o := NewOpponent(Ghost)
		o.rng = rand.New(rand.NewSource(seed))
		o.X, o.Y = 10, 10
		room.Enemies = []*Opponent{o}
		for i := 0; i < 20; i++ {
			o.Move(hero.X, hero.Y, exit, rooms, nil, room.Enemies)
			if (Point{o.X, o.Y}) == exit {
				t.Fatalf("seed %d: Riki blinked onto the portal", seed)
			}
			if (Point{o.X, o.Y}) == (Point{10, 10}) {
				t.Fatalf("seed %d: Riki blinked onto his own cell", seed)
			}
			o.X, o.Y = 10, 10
		}
	}
}

// A blink needs a room: in a corridor Riki used to stand still for good.
func TestRikiWalksOutOfACorridor(t *testing.T) {
	corridor := []Rect{{X: 20, Y: 9, W: 10, H: 3}} // centre line y=10, x=21..28
	for seed := int64(0); seed < 50; seed++ {
		o := NewOpponent(Ghost)
		o.rng = rand.New(rand.NewSource(seed))
		o.X, o.Y = 24, 10
		o.Move(60, 60, Point{-1, -1}, nil, corridor, []*Opponent{o})
		if (Point{o.X, o.Y}) == (Point{24, 10}) || !InPassageCenter(o.X, o.Y, corridor) {
			t.Fatalf("seed %d: Riki at %d,%d, want one step along the corridor", seed, o.X, o.Y)
		}
	}
}

// No enemy walks, steps twice or blinks onto the portal.
func TestEnemiesNeverStandOnThePortal(t *testing.T) {
	for seed := int64(1); seed <= 40; seed++ {
		s := NewSessionSeed(seed)
		for turn := 0; turn < 1000 && s.Player.IsAlive() && !s.Won(); turn++ {
			if err := s.ApplyAction(testAction(s)); err != nil {
				t.Fatal(err)
			}
			if o := s.OpponentAt(s.Level.Exit.X, s.Level.Exit.Y); o != nil {
				t.Fatalf("seed %d turn %d: %s stands on the portal", seed, turn, o.Type.DisplayName())
			}
		}
	}
}

// Riki used to show on every new level until the enemies' first turn.
func TestRikiStartsHidden(t *testing.T) {
	for _, kind := range AllOpponentTypes {
		if got := NewOpponent(kind).IsVisible; got != (kind != Ghost) {
			t.Fatalf("%s starts visible=%v", kind.DisplayName(), got)
		}
	}
}
