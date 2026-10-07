package domain

import (
	"math/rand"
	"testing"
)

// Riki прыгает в случайную клетку своей комнаты; клетка героя такой быть не должна.
func TestRikiNeverBlinksOntoTheHero(t *testing.T) {
	room := &Room{X: 10, Y: 10, W: 4, H: 1}
	rooms := []*Room{room}
	hero := Point{13, 10} // дальше радиуса агрессии Riki
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
			o.X, o.Y = 10, 10 // держим его вне досягаемости, чтобы он продолжал прыгать
		}
	}
}

// Ни один враг, как бы он ни ходил, не должен стоять в клетке героя.
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

// Раньше прыжок Riki выбирал любую свободную клетку комнаты, включая портал.
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

// Прыжку нужна комната: в коридоре Riki раньше стоял на месте навсегда.
func TestRikiWalksOutOfACorridor(t *testing.T) {
	corridor := []Rect{{X: 20, Y: 9, W: 10, H: 3}} // центральная линия y=10, x=21..28
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

// Ни один враг не заходит на портал ни шагом, ни двойным шагом, ни прыжком.
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

// Раньше Riki был виден на каждом новом уровне до первого хода врагов.
func TestRikiStartsHidden(t *testing.T) {
	for _, kind := range AllOpponentTypes {
		if got := NewOpponent(kind).IsVisible; got != (kind != Ghost) {
			t.Fatalf("%s starts visible=%v", kind.DisplayName(), got)
		}
	}
}
