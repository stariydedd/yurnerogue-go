package domain

import (
	"reflect"
	"testing"
)

func TestSpecialStrikeDamageCooldownAndInvalidTargets(t *testing.T) {
	s, enemy := combatEventSession(Zombie)
	s.Player.Agility = 1000000
	baseDamage := InitialDamage + int(float64(s.Player.Strength-StandardStrength)*StrengthFactor)
	enemy.Health = 10000
	if err := s.ApplyAction("tw"); err == nil || s.Turns != 0 || s.Actions() != "" || s.Player.StrikeCooldown != 0 {
		t.Fatal("empty strike spent a turn")
	}
	if err := s.ApplyAction("td"); err != nil {
		t.Fatal(err)
	}
	if s.CombatEvents[0].Damage != (baseDamage*3+1)/2 || !s.CombatEvents[0].Critical || s.Player.StrikeCooldown != StrikeRechargeAttacks || s.Stats.AttacksMade != 1 || s.Turns != 1 {
		t.Fatalf("invalid strike: %+v %+v", s.CombatEvents, s.Player)
	}
	if s.ApplyAction("td") == nil || s.Turns != 1 {
		t.Fatal("cooldown bypassed")
	}
	for i := 0; i < StrikeRechargeAttacks; i++ {
		if err := s.ApplyAction("d"); err != nil {
			t.Fatal(err)
		}
	}
	if s.Player.StrikeCooldown != 0 || s.CombatEvents[0].Damage != baseDamage || s.CombatEvents[0].Critical {
		t.Fatal("boost leaked into ordinary attacks")
	}
	if err := s.ApplyAction("td"); err != nil {
		t.Fatal(err)
	}
	for _, action := range []string{"t", "tb", "tW", "tt", "tz"} {
		if s.ApplyAction(action) == nil {
			t.Fatalf("accepted %q", action)
		}
	}
}

func TestWalkingDoesNotRechargeSpecialStrike(t *testing.T) {
	s, enemy := combatEventSession(Zombie)
	s.Player.Agility = 1000000
	enemy.Health = 10000
	if err := s.ApplyAction("td"); err != nil {
		t.Fatal(err)
	}
	for _, room := range s.Level.Rooms {
		room.Enemies = nil // nothing left to hit: only walking remains
	}
	for i := 0; i < 10; i++ {
		s.Player.Health = s.Player.MaxHealth
		if err := s.ApplyAction("a"); err != nil {
			t.Fatal(err)
		}
		if err := s.ApplyAction("d"); err != nil {
			t.Fatal(err)
		}
	}
	if s.Player.StrikeCooldown != StrikeRechargeAttacks {
		t.Fatalf("walking recharged the strike to %d", s.Player.StrikeCooldown)
	}
	if s.ApplyAction("tw") == nil || s.Message != "Critical Strike charges with attacks: 3 left." {
		t.Fatalf("charge message %q", s.Message)
	}
}

func TestDefenseNeedsAnEnemyInContact(t *testing.T) {
	s, enemy := combatEventSession(Zombie)
	enemy.X, enemy.Y = s.Player.X+2, s.Player.Y // approaching, not yet in contact
	turns := s.Turns
	if s.ApplyAction("b") == nil || s.Turns != turns || s.Actions() != "" || s.Player.GuardCooldown != 0 {
		t.Fatal("defense worked as a free wait")
	}
	if s.Message != "No enemy next to you." {
		t.Fatalf("message %q", s.Message)
	}
	snake, other := combatEventSession(Snake)
	other.X, other.Y = snake.Player.X+1, snake.Player.Y+1 // Skywrath hits diagonally
	if err := snake.ApplyAction("b"); err != nil {
		t.Fatal("defense must work against a diagonal Skywrath")
	}
}

func TestParryBlocksHitsCompletelyAndExpires(t *testing.T) {
	for _, kind := range []OpponentType{Zombie, Vampire, Ogre, Snake} {
		s, enemy := combatEventSession(kind)
		s.Player.Agility = -1000000
		enemy.Strength = 100
		if kind == Snake {
			enemy.X, enemy.Y = s.Player.X+1, s.Player.Y+1
			enemy.Health = 1_000_000 // survive every riposte of the loop below
		}
		health, maxHealth := s.Player.Health, s.Player.MaxHealth
		for i := 0; i < 50 && kind == Snake; i++ {
			// A parried Skywrath hit must never put the hero to sleep.
			s.Player.GuardCooldown = 0
			if err := s.ApplyAction("b"); err != nil {
				t.Fatalf("parry against Skywrath failed on try %d: %v", i, err)
			}
			if s.Player.Sleeping {
				t.Fatalf("parried Skywrath hit caused sleep on try %d", i)
			}
		}
		if kind == Snake {
			continue
		}
		if err := s.ApplyAction("b"); err != nil {
			t.Fatal(err)
		}
		e := s.CombatEvents[0]
		if !e.Parried || e.Damage != 0 || !e.TargetPlayer || s.Player.Health != health || s.Player.MaxHealth != maxHealth ||
			s.Player.Guarding || s.Player.GuardCooldown != GuardRechargeHits || s.Stats.HitsTaken != 0 {
			t.Fatalf("%v: parry let damage through: %+v", kind, s.CombatEvents)
		}
		if s.ApplyAction("b") == nil || s.Turns != 1 {
			t.Fatal("defense spam spends turns")
		}
		if s.Message != "Parry charges with hits taken: 3 left." {
			t.Fatalf("charge message %q", s.Message)
		}
		// Only hits landed after the parry turn charge it again.
		enemy.Health = 1_000_000
		for turn := 0; s.Player.GuardCooldown > 0; turn++ {
			if turn > 10 {
				t.Fatalf("%v: parry did not recharge from hits", kind)
			}
			hits, cooldown := s.Stats.HitsTaken, s.Player.GuardCooldown
			s.Player.Health = s.Player.MaxHealth
			if err := s.ApplyAction("d"); err != nil {
				t.Fatal(err)
			}
			if landed := s.Stats.HitsTaken - hits; s.Player.GuardCooldown != max(0, cooldown-landed) {
				t.Fatalf("%v: %d hits changed charge %d -> %d", kind, landed, cooldown, s.Player.GuardCooldown)
			}
		}
	}
}

func TestAutomaticEquipmentAndScrolls(t *testing.T) {
	s, _ := combatEventSession(Zombie)
	p := s.Player
	for i := 0; i < MaxBackpackItemsPerType; i++ {
		p.PickUpItem(&Item{Type: ItemScroll})
	}
	weapon := &Item{Type: ItemWeapon, Name: "Yasha", StrengthEffect: 35, X: p.X, Y: p.Y}
	s.Level.Items = []*Item{weapon}
	s.CheckItemPickup()
	if p.Weapon != weapon || len(s.Level.Items) != 0 {
		t.Fatal("weapon not equipped")
	}
	// A weapon that is not stronger always sharpens the equipped one by +1,
	// with no ceiling, and never replaces it.
	for _, tc := range []struct{ found, want int }{{34, 36}, {4, 37}, {37, 38}, {1, 39}} {
		s.Level.Items = []*Item{{Type: ItemWeapon, Name: "Diffusal Blade", StrengthEffect: tc.found, X: p.X, Y: p.Y}}
		s.CheckItemPickup()
		if p.Weapon != weapon || weapon.StrengthEffect != tc.want || len(s.Level.Items) != 0 || len(p.Backpack) != MaxBackpackItemsPerType {
			t.Fatalf("found +%d: weapon +%d, want +%d", tc.found, weapon.StrengthEffect, tc.want)
		}
	}
	s.Level.Items = []*Item{{Type: ItemWeapon, Name: "Diffusal Blade", StrengthEffect: 20, X: p.X, Y: p.Y}}
	s.CheckItemPickup()
	if weapon.StrengthEffect != 40 || weapon.Name != "Butterfly" || s.Message != "Sharpened Butterfly." {
		t.Fatalf("sharpening: %s +%d %q", weapon.Name, weapon.StrengthEffect, s.Message)
	}
	// Crossing into the next price tier renames the weapon.
	blade := &Item{Type: ItemWeapon, Name: "Shadow Blade", StrengthEffect: 20}
	p.Weapon = blade
	s.Level.Items = []*Item{{Type: ItemWeapon, Name: "Crystalys", StrengthEffect: 3, X: p.X, Y: p.Y}}
	s.CheckItemPickup()
	if blade.StrengthEffect != 21 || blade.Name != "Desolator" || s.Message != "Picked up: Desolator." {
		t.Fatalf("tier change: %s +%d %q", blade.Name, blade.StrengthEffect, s.Message)
	}
	// Any weapon, even +1 at high strength, replaces the starter blade.
	p.Weapon = nil
	p.Strength = 200
	weak := &Item{Type: ItemWeapon, Name: "Yasha", StrengthEffect: 1, X: p.X, Y: p.Y}
	s.Level.Items = []*Item{weak}
	s.CheckItemPickup()
	if p.Weapon != weak || WeaponDamage(p, weak) <= WeaponDamage(p, nil) {
		t.Fatal("a +1 weapon must replace and outdamage the starter")
	}
	p.Strength = DefaultStrength
	p.Health = 100
	before := p.MaxHealth
	s.Level.Items = []*Item{{Type: ItemScroll, MaxHealthEffect: 7, X: p.X, Y: p.Y}}
	s.CheckItemPickup()
	s.CheckItemPickup()
	if p.MaxHealth != before+7 || p.Health != 107 || s.Stats.ScrollsRead != 1 || len(p.Backpack) != MaxBackpackItemsPerType {
		t.Fatal("scroll must apply exactly once regardless of backpack capacity")
	}
}

func TestParryIgnoresHitsDuringItsOwnTurnAndWalking(t *testing.T) {
	s, first := combatEventSession(Zombie)
	second := NewOpponent(Zombie)
	second.X, second.Y = s.Player.X, s.Player.Y+1
	s.Level.Rooms[0].Enemies = append(s.Level.Rooms[0].Enemies, second)
	s.Player.Agility = -1_000_000
	first.Health, second.Health = 1_000_000, 1_000_000
	if err := s.ApplyAction("b"); err != nil {
		t.Fatal(err)
	}
	parried := 0
	for _, e := range s.CombatEvents {
		if e.Parried {
			parried++
		}
	}
	if parried != 2 || s.Stats.HitsTaken != 0 || s.Player.GuardCooldown != GuardRechargeHits {
		t.Fatalf("two parried hits changed the charge: %d", s.Player.GuardCooldown)
	}
	for _, room := range s.Level.Rooms {
		room.Enemies = nil
	}
	for i := 0; i < 10; i++ {
		if err := s.ApplyAction("a"); err != nil {
			t.Fatal(err)
		}
	}
	if s.Player.GuardCooldown != GuardRechargeHits {
		t.Fatal("walking recharged the parry")
	}
}

func TestParryStrikesBackOnlyWhenHit(t *testing.T) {
	// Hit: blocked completely, attacker struck back for half a normal hit.
	for _, kind := range []OpponentType{Zombie, Vampire, Ogre} {
		s, enemy := combatEventSession(kind)
		s.Player.Agility = -1_000_000 // enemies never miss
		enemy.Health = 10_000
		counter := WeaponDamage(s.Player, nil) * RiposteDamagePercent / 100
		if err := s.ApplyAction("b"); err != nil {
			t.Fatal(err)
		}
		if enemy.Health != 10_000-counter || len(s.CombatEvents) != 2 || s.CombatEvents[1].TargetPlayer || s.CombatEvents[1].Damage != counter {
			t.Fatalf("%v: no riposte, health %d events %+v", kind, enemy.Health, s.CombatEvents)
		}
		if s.Stats.AttacksMade != 0 || s.Message != "You parried the "+enemy.Type.DisplayName()+" and struck back for "+itoa(counter)+" dmg." {
			t.Fatalf("%v: riposte counted as an attack or wrong message %q", kind, s.Message)
		}
	}
	// Miss: nothing to strike back at.
	s, enemy := combatEventSession(Zombie)
	s.Player.Agility = 1_000_000
	before := enemy.Health
	if err := s.ApplyAction("b"); err != nil || enemy.Health != before {
		t.Fatal("riposte after a miss")
	}
	// Resting Axe does not attack, so there is no riposte either.
	s, enemy = combatEventSession(Ogre)
	enemy.SetResting(true)
	before = enemy.Health
	if err := s.ApplyAction("b"); err != nil || enemy.Health != before {
		t.Fatal("riposte against a resting Axe")
	}
	// A lethal riposte pays gold and counts the kill.
	s, enemy = combatEventSession(Zombie)
	s.Player.Agility = -1_000_000
	enemy.Health = 1
	if err := s.ApplyAction("b"); err != nil || enemy.IsAlive() || s.Stats.EnemiesKilled != 1 || s.Player.Treasures == 0 {
		t.Fatal("lethal riposte must kill and pay gold")
	}
}

func TestSleepingAbilitiesDoNotActivate(t *testing.T) {
	for _, action := range []string{"td", "b"} {
		s, _ := combatEventSession(Zombie)
		s.Player.FallAsleep(1)
		if err := s.ApplyAction(action); err != nil {
			t.Fatal(err)
		}
		if s.Turns != 1 || s.Stats.AttacksMade != 0 || s.Player.StrikeCooldown != 0 || s.Player.GuardCooldown != 0 || s.Player.Guarding {
			t.Fatal("ability activated while asleep")
		}
	}
}

func TestAbilitiesReplayToSameResult(t *testing.T) {
	s := NewSessionSeed(1)
	usedStrike, usedGuard := false, false
	for i := 0; i < 5000 && s.Player.IsAlive() && !s.Won(); i++ {
		a := testAction(s)
		if !s.Player.Sleeping {
			if !usedGuard && s.enemyInContact() {
				a = "b"
				usedGuard = true
			} else if s.Player.StrikeCooldown == 0 {
				for j, d := range dirs4 {
					if s.OpponentAt(s.Player.X+d.X, s.Player.Y+d.Y) != nil {
						a = "t" + string("wsad"[j])
						usedStrike = true
						break
					}
				}
			}
		}
		if err := s.ApplyAction(a); err != nil {
			t.Fatal(err)
		}
	}
	replayed, err := Replay(1, s.Actions())
	if err != nil {
		t.Fatal(err)
	}
	if !usedGuard || !usedStrike || !reflect.DeepEqual(s.Stats, replayed.Stats) || s.Player.Treasures != replayed.Player.Treasures {
		t.Fatal("ability replay diverged")
	}
}

func TestWaitSkipsATurnAndLetsEnemiesApproach(t *testing.T) {
	s, enemy := combatEventSession(Zombie)
	enemy.X, enemy.Y = s.Player.X+3, s.Player.Y // chasing, not yet in contact
	before := abs(enemy.X-s.Player.X) + abs(enemy.Y-s.Player.Y)
	x, y := s.Player.X, s.Player.Y
	if err := s.ApplyAction("z"); err != nil {
		t.Fatal(err)
	}
	after := abs(enemy.X-s.Player.X) + abs(enemy.Y-s.Player.Y)
	if s.Turns != 1 || s.Actions() != "z" || s.Player.X != x || s.Player.Y != y || after >= before {
		t.Fatalf("wait: turns %d actions %q enemy %d -> %d", s.Turns, s.Actions(), before, after)
	}
	if s.Stats.TilesMoved != 0 || s.Stats.AttacksMade != 0 {
		t.Fatal("waiting counted as a step or an attack")
	}
}

func TestWeaponLogSaysPickedUpOnlyWhenTheNameChanges(t *testing.T) {
	s, _ := combatEventSession(Zombie)
	p := s.Player
	pick := func(name string, bonus int) string {
		s.Level.Items = []*Item{{Type: ItemWeapon, Name: name, StrengthEffect: bonus, X: p.X, Y: p.Y}}
		s.CheckItemPickup()
		return s.Message
	}
	for i, tc := range []struct {
		name  string
		bonus int
		want  string
	}{
		{"Yasha", 7, "Picked up: Yasha."},           // starter blade -> first weapon
		{"Yasha", 9, "Sharpened Yasha."},            // stronger, but the same name
		{"Crystalys", 2, "Sharpened Yasha."},        // weaker: +1 to Yasha
		{"Butterfly", 38, "Picked up: Butterfly."},  // stronger with a new name
		{"Crystalys", 1, "Sharpened Butterfly."},    // +39 is still Butterfly
		{"Crystalys", 1, "Sharpened Butterfly."},    // +40 is still Butterfly
		{"Crystalys", 1, "Picked up: Silver Edge."}, // +41 crosses into the next name
	} {
		if got := pick(tc.name, tc.bonus); got != tc.want {
			t.Fatalf("step %d: %q, want %q", i, got, tc.want)
		}
	}
}
