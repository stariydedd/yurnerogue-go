package domain

import "testing"

// runPast ставит героя в клетку 10,10 боевой комнаты и запускает бег вправо;
// враг стоит прямо над второй клеткой пути.
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

// Раньше бег останавливался только при потере здоровья и пробегал мимо удара
// в щит, промаха и кражи, после которой здоровье и так ниже максимума.
func TestRunStopsWhenAnEnemyAttacks(t *testing.T) {
	s, _ := runPast(t, 1_000_000, 10_000) // каждый удар попадает в щит
	if s.Player.X != 11 || s.Player.Health != DefaultMaxHealth {
		t.Fatalf("a hit taken by the shield: hero at x=%d, health %d", s.Player.X, s.Player.Health)
	}
	s, _ = runPast(t, -1_000_000, 0) // каждый удар мимо
	if s.Player.X != 11 {
		t.Fatalf("a miss: hero ran on to x=%d", s.Player.X)
	}
}

func TestRunStopsOnAnyItem(t *testing.T) {
	for _, it := range []*Item{
		{Type: ItemScroll, StrengthEffect: 2},                // читается при подборе
		{Type: ItemWeapon, Name: "Yasha", StrengthEffect: 9}, // экипируется при подборе
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
