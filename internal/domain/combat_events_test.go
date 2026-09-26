package domain

import "testing"

func combatEventSession(kind OpponentType) (*Session, *Opponent) {
	s := NewSessionSeed(21)
	s.Player.X, s.Player.Y = 12, 10
	o := NewOpponent(kind)
	o.X, o.Y = 13, 10
	s.Level.Rooms = []*Room{{X: 8, Y: 6, W: 16, H: 9, Enemies: []*Opponent{o}}}
	s.Level.Items, s.Level.Passages = nil, nil
	s.Level.Exit = Point{X: 20, Y: 12}
	return s, o
}

func TestCombatEventIncludesLethalHitAndResetsEachAction(t *testing.T) {
	s, o := combatEventSession(Zombie)
	s.Player.Agility = 1_000_000
	o.Health = 1
	if err := s.ApplyAction("d"); err != nil {
		t.Fatal(err)
	}
	if len(s.CombatEvents) != 1 {
		t.Fatalf("lethal attack should produce one event: %+v", s.CombatEvents)
	}
	e := s.CombatEvents[0]
	if e.Target != (Point{13, 10}) || e.TargetPlayer || e.Damage <= 0 || e.Level != 1 || o.IsAlive() {
		t.Fatalf("wrong lethal event: %+v", e)
	}
	if err := s.ApplyAction("d"); err != nil || len(s.CombatEvents) != 0 {
		t.Fatal("walking repeats the previous hit")
	}
	s.recordCombat(e)
	if s.ApplyAction("invalid") == nil || len(s.CombatEvents) != 0 {
		t.Fatal("invalid action retains stale combat events")
	}
}

func TestCombatEventsKeepBothSidesAndMisses(t *testing.T) {
	s, o := combatEventSession(Zombie)
	s.Player.Agility = -1_000_000
	o.Agility = 1_000_000
	before := s.Player.Health
	if err := s.ApplyAction("d"); err != nil {
		t.Fatal(err)
	}
	if len(s.CombatEvents) != 2 {
		t.Fatalf("expected miss and counterattack: %+v", s.CombatEvents)
	}
	miss, hit := s.CombatEvents[0], s.CombatEvents[1]
	if miss.Damage != Miss || miss.TargetPlayer || hit.Damage != before-s.Player.Health || !hit.TargetPlayer || hit.Target != (Point{12, 10}) {
		t.Fatalf("wrong attack results: %+v", s.CombatEvents)
	}
	s.CombatEvents = nil
	s.Player.Agility, o.Agility = 1_000_000, -1_000_000
	s.ProcessEnemyTurns()
	if len(s.CombatEvents) != 1 || s.CombatEvents[0].Damage != Miss || !s.CombatEvents[0].TargetPlayer {
		t.Fatal("enemy miss was not reported")
	}
}

func TestCombatEventsDistinguishRestDrainAndZeroDamage(t *testing.T) {
	s, o := combatEventSession(Ogre)
	o.SetResting(true)
	s.ProcessEnemyTurns()
	if len(s.CombatEvents) != 0 {
		t.Fatal("resting Axe produced a hit or miss")
	}
	s, o = combatEventSession(Vampire)
	o.Agility = 1_000_000
	s.Player.Health = 1 // max-HP drain need not change current HP.
	before := s.Player.MaxHealth
	s.ProcessEnemyTurns()
	if len(s.CombatEvents) != 1 || !s.CombatEvents[0].MaxHP || s.CombatEvents[0].Damage != before-s.Player.MaxHealth {
		t.Fatalf("max HP drain lost: %+v", s.CombatEvents)
	}
	s, _ = combatEventSession(Zombie)
	s.Player.Agility, s.Player.Strength = 1_000_000, -1_000_000
	if err := s.ApplyAction("d"); err != nil {
		t.Fatal(err)
	}
	if len(s.CombatEvents) < 1 || s.CombatEvents[0].Damage != 0 {
		t.Fatal("zero-damage hit mislabeled as a miss")
	}
}

func TestCombatEventsAreBoundedAndSnapshotCoordinates(t *testing.T) {
	s, _ := combatEventSession(Zombie)
	for i := 0; i < 1000; i++ {
		s.recordCombat(CombatEvent{Target: Point{12, 10}, Damage: i})
	}
	s.Player.X++
	if len(s.CombatEvents) != maxCombatEvents || s.CombatEvents[0].Damage != 1000-maxCombatEvents || s.CombatEvents[maxCombatEvents-1].Target != (Point{12, 10}) {
		t.Fatal("unbounded queue or impact position changed")
	}
}

func TestCombatEventsKeepEveryCounterattackInOneTurn(t *testing.T) {
	s, first := combatEventSession(Zombie)
	first.Agility = 1_000_000
	second := NewOpponent(Zombie)
	second.X, second.Y, second.Agility = 12, 11, -1_000_000
	s.Level.Rooms[0].Enemies = append(s.Level.Rooms[0].Enemies, second)
	s.ProcessEnemyTurns()
	if len(s.CombatEvents) != 2 || s.CombatEvents[0].Damage <= 0 || s.CombatEvents[1].Damage != Miss {
		t.Fatalf("one attack replaced another: %+v", s.CombatEvents)
	}
	for _, e := range s.CombatEvents {
		if !e.TargetPlayer || e.Target != (Point{12, 10}) {
			t.Fatalf("wrong counterattack target: %+v", e)
		}
	}
}

func TestHitsTheShieldTookWholeAreMarked(t *testing.T) {
	shielded := func(kind OpponentType, shield int) (*Session, CombatEvent) {
		s, o := combatEventSession(kind)
		s.Player.Agility, o.Agility = -1_000_000, 1_000_000
		if shield > 0 {
			clarity := &Item{Type: ItemElixir, MaxHealthEffect: shield}
			s.Player.PickUpItem(clarity)
			s.Player.UseItem(clarity)
		}
		s.ProcessEnemyTurns()
		if len(s.CombatEvents) != 1 {
			t.Fatalf("%v: want one hit, got %+v", kind, s.CombatEvents)
		}
		return s, s.CombatEvents[0]
	}

	s, e := shielded(Zombie, 10_000)
	if !e.Shielded || e.Damage <= 0 || s.Player.Health != DefaultMaxHealth {
		t.Fatalf("a hit the shield took whole: %+v, health %d", e, s.Player.Health)
	}
	if _, e = shielded(Zombie, 1); e.Shielded {
		t.Fatalf("a hit that got past the shield is marked: %+v", e)
	}
	if _, e = shielded(Zombie, 0); e.Shielded {
		t.Fatalf("a hit with no shield is marked: %+v", e)
	}
	s, e = shielded(Vampire, 10_000)
	if !e.Shielded || e.MaxHP || s.Player.MaxHealth != DefaultMaxHealth || s.Message != "The Bloodseeker struck your shield." {
		t.Fatalf("a drain the shield took whole: %+v, %q", e, s.Message)
	}
}
