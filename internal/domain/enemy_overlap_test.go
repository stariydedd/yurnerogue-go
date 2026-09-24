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
			o.Move(hero.X, hero.Y, rooms, nil, room.Enemies)
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
