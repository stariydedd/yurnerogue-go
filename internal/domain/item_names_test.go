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

func TestWeaponNamesPreserveVersionOneRandomStream(t *testing.T) {
	seen := map[string]bool{}
	for seed := int64(0); seed < 100; seed++ {
		current, legacy := rand.New(rand.NewSource(seed)), rand.New(rand.NewSource(seed))
		for i := 0; i < 100; i++ {
			legacy.Intn(9)
			wantDamage := legacy.Intn(21) + 30
			item := NewWeapon(current)
			seen[item.Name] = true
			if item.StrengthEffect != wantDamage || current.Int63() != legacy.Int63() {
				t.Fatalf("seed %d weapon %d changed simulation", seed, i)
			}
		}
	}
	for _, name := range weaponNames {
		if !seen[name] {
			t.Fatalf("weapon %s is unreachable", name)
		}
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
