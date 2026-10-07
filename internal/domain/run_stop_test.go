package domain

import "testing"

// runPast puts the hero at 10,10 of the combat room, running right, with an
// enemy standing just above the second cell of the way.
func runPast(t *testing.T, agility int, shield int) (*Session, *Opponent) {
	t.Helper()
	s, o := combatEventSession(Zombie)
	s.Player.X, s.Player.Y = 10, 10
	o.X, o.Y, o.Agility, o.Health = 11, 9, agility, 1_000_000
	if shield > 0 {
		clarity := &Item{Type: ItemElixir, MaxHealthEffect: shield}
		s.Player.PickUpItem(clarity)
		s.Player.UseItem(clarity)
	}
	if err := s.ApplyAction("D"); err != nil {
		t.Fatal(err)
	}
	return s, o
}

// The run used to stop only when health dropped, so it ran on through a hit
// the shield took, a miss, or a drain that left health below the maximum.
func TestRunStopsWhenAnEnemyAttacks(t *testing.T) {
	s, _ := runPast(t, 1_000_000, 10_000) // every hit lands in the shield
	if s.Player.X != 11 || s.Player.Health != DefaultMaxHealth {
		t.Fatalf("a hit taken by the shield: hero at x=%d, health %d", s.Player.X, s.Player.Health)
	}
	s, _ = runPast(t, -1_000_000, 0) // every hit misses
	if s.Player.X != 11 {
		t.Fatalf("a miss: hero ran on to x=%d", s.Player.X)
	}
}

func TestRunStopsOnAnyItem(t *testing.T) {
	for _, it := range []*Item{
		{Type: ItemScroll, StrengthEffect: 2},                // read on pickup
		{Type: ItemWeapon, Name: "Yasha", StrengthEffect: 9}, // equipped on pickup
	} {
		s, o := combatEventSession(Zombie)
		o.Health = 0
		s.Player.X, s.Player.Y = 10, 10
		it.X, it.Y = 12, 10
		s.Level.Items = []*Item{it}
		if err := s.ApplyAction("D"); err != nil {
			t.Fatal(err)
		}
		if s.Player.X != 12 {
			t.Fatalf("type %d: the run went on to x=%d over the item", it.Type, s.Player.X)
		}
	}
}
