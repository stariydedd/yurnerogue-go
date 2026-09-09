package domain

import "testing"

func TestEffectStatusesAreIndependentSnapshots(t *testing.T) {
	p := NewPerson()
	if len(p.EffectStatuses()) != 0 {
		t.Fatal("new person has effects")
	}
	item := &Item{Type: ItemElixir, StrengthEffect: 7, AgilityEffect: 3}
	p.PickUpItem(item)
	p.UseItem(item)
	before := p.EffectStatuses()
	if len(before) != 2 || before[0].TurnsLeft != ElixirDuration {
		t.Fatalf("unexpected effects: %+v", before)
	}
	before[0].Amount = 999
	if p.EffectStatuses()[0].Amount == 999 {
		t.Fatal("snapshot changed live effect")
	}
	p.TickEffects()
	if before[0].TurnsLeft != ElixirDuration {
		t.Fatal("live tick mutated snapshot")
	}
	if p.EffectStatuses()[0].TurnsLeft != ElixirDuration-1 {
		t.Fatal("snapshot did not reflect new tick")
	}
	for i := 1; i < ElixirDuration; i++ {
		p.TickEffects()
	}
	if len(p.EffectStatuses()) != 0 || p.Strength != DefaultStrength || p.Agility != DefaultAgility {
		t.Fatal("effect expiry changed")
	}
}
