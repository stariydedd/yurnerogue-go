package domain

import (
	"math/rand"
	"strings"
	"testing"
	"unicode/utf8"
)

func TestItemNamesAreCompactAndKeepReplayChoiceCounts(t *testing.T) {
	for category, names := range map[string][]string{"food": foodNames, "elixir": elixirNames, "scroll": scrollNames, "weapon": weaponNames} {
		wantCount := 9
		if category == "weapon" {
			wantCount = 10
		}
		if len(names) != wantCount {
			t.Fatalf("%s has %d variants, want %d", category, len(names), wantCount)
		}
		seen := map[string]bool{}
		for _, name := range names {
			if strings.TrimSpace(name) != name || name == "" || utf8.RuneCountInString(name) > 17 {
				t.Fatalf("%s name is not compact: %q", category, name)
			}
			if seen[name] {
				t.Fatalf("duplicate %s name: %q", category, name)
			}
			seen[name] = true
		}
	}
}

func TestWeaponsScaleWithDepthAndReachEveryName(t *testing.T) {
	seen := map[string]bool{}
	for level := 1; level <= MaxLevels; level++ {
		low, high := WeaponBonusRange(level)
		rolled := map[int]bool{}
		rng := rand.New(rand.NewSource(int64(level)))
		for i := 0; i < 2000; i++ {
			item := NewWeapon(level, rng)
			seen[item.Name] = true
			if item.StrengthEffect < low || item.StrengthEffect > high {
				t.Fatalf("level %d weapon +%d outside %d..%d", level, item.StrengthEffect, low, high)
			}
			rolled[item.StrengthEffect] = true
		}
		if len(rolled) != high-low+1 {
			t.Fatalf("level %d does not reach every bonus", level)
		}
	}
	if low, high := WeaponBonusRange(1); low != 1 || high != 17 {
		t.Fatal("first-level weapons must start from +1")
	}
	if low, high := WeaponBonusRange(MaxLevels); low != 36 || high != MaxWeaponBonus {
		t.Fatal("last-level weapons must reach +50")
	}
	for level := 1; level <= MaxLevels+5; level++ {
		if _, high := WeaponBonusRange(level); high > MaxWeaponBonus {
			t.Fatalf("level %d exceeds +%d", level, MaxWeaponBonus)
		}
	}
	p := NewPerson()
	if WeaponDamage(p, &Item{Type: ItemWeapon, StrengthEffect: 1}) <= WeaponDamage(p, nil) {
		t.Fatal("even a +1 weapon must beat the starter")
	}
	if got := WeaponDamage(p, &Item{Type: ItemWeapon, StrengthEffect: 50}); got != 94 {
		t.Fatalf("+50 at starting strength deals %d, want 94", got)
	}
	for _, name := range weaponNames {
		if !seen[name] {
			t.Fatalf("weapon %s is unreachable", name)
		}
	}
}

func TestWeaponNameFollowsBonusByDotaCost(t *testing.T) {
	for bonus, want := range map[int]string{
		1: "Crystalys", 5: "Crystalys", 6: "Yasha", 10: "Yasha", 11: "Diffusal Blade",
		25: "Desolator", 26: "Battle Fury", 45: "Silver Edge", 46: "Abyssal Blade",
		50: "Abyssal Blade", 0: "Crystalys", 77: "Abyssal Blade",
	} {
		if got := WeaponName(bonus); got != want {
			t.Fatalf("+%d is %s, want %s", bonus, got, want)
		}
	}
	for level := 1; level <= MaxLevels; level++ {
		item := NewWeapon(level)
		if item.Name != WeaponName(item.StrengthEffect) {
			t.Fatalf("generated %s +%d does not match its bonus", item.Name, item.StrengthEffect)
		}
	}
}

func TestScrollElixirAndFoodNeverRollTiny(t *testing.T) {
	p := NewSessionSeed(7).Player
	for level := 1; level <= MaxLevels; level++ {
		b := StatBonusRange(level)
		for i := 0; i < 300; i++ {
			for multiplier, item := range map[int]*Item{1: NewScroll(p, level), 2: NewElixir(p, level)} {
				value, low, high := item.StrengthEffect+item.AgilityEffect, b.StatLow, b.StatHigh
				if item.SubType == SubHealth {
					value, low, high = item.MaxHealthEffect, b.HealthLow, b.HealthHigh
				}
				if value < low*multiplier || value > high*multiplier || value < 2 {
					t.Fatalf("level %d %s rolled %d outside %d..%d", level, item.Name, value, low*multiplier, high*multiplier)
				}
			}
		}
	}
	if b := StatBonusRange(1); b != (StatBonus{StatLow: 2, StatHigh: 3, HealthLow: 13, HealthHigh: 25}) {
		t.Fatalf("first-level scrolls: %+v", b)
	}
	if b := StatBonusRange(MaxLevels); b != (StatBonus{StatLow: 6, StatHigh: 9, HealthLow: 63, HealthHigh: 125}) {
		t.Fatalf("last-level scrolls: %+v", b)
	}
	// Food depends on depth only: a drained max HP must not shrink it.
	p.MaxHealth = 50
	for level := 1; level <= MaxLevels; level++ {
		low, high := FoodHealRange(level)
		for i := 0; i < 200; i++ {
			if heal := NewFood(p, level).HealthEffect; heal < low || heal > high {
				t.Fatalf("level %d food heals %d, want %d..%d", level, heal, low, high)
			}
		}
	}
	if low, high := FoodHealRange(1); low != 25 || high != 50 {
		t.Fatal("first-level food must heal 25..50")
	}
	if low, high := FoodHealRange(MaxLevels); low != 75 || high != 150 {
		t.Fatal("last-level food must heal 75..150")
	}
}

func TestBaseWeaponReturnsWithoutDroppingOrConsumingSlots(t *testing.T) {
	s := NewSessionSeed(21)
	p := s.Player
	if p.Weapon != nil || p.AttackStrength() != p.Strength {
		t.Fatal("starter changed initial strength")
	}
	weapon := &Item{Type: ItemWeapon, Name: "Yasha", StrengthEffect: 35}
	p.PickUpItem(weapon)
	groundCount := len(s.Level.Items)
	if !s.UseChoice(ItemWeapon, 1) || p.Weapon != weapon || len(s.Level.Items) != groundCount || len(p.Backpack) != 0 {
		t.Fatal("equipping upgrade dropped the starter")
	}
	if !s.UseChoice(ItemWeapon, 0) || p.Weapon != nil || len(p.Backpack) != 1 || p.Backpack[0] != weapon || len(s.Level.Items) != groundCount {
		t.Fatal("returning to starter lost or duplicated a weapon")
	}
}
