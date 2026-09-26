package render

import (
	"image"
	"image/color"
	"testing"

	"github.com/stariydedd/yurnerogue-go/internal/domain"
)

func TestCombatLabelsDoNotOverlapOrEnterHUD(t *testing.T) {
	for _, layout := range []Layout{DesktopLayout(), TouchLayout(390, 700)} {
		field := image.Rect(0, 0, layout.GridW, layout.GridH)
		for _, y := range []int{2, layout.GridH / 2, layout.GridH - 14} {
			var placed []image.Rectangle
			for i := 0; i < 8; i++ {
				want := image.Rect(90+i%2*32, y, 180+i%2*32, y+12)
				got, ok := placeCombatLabel(want, field, placed)
				if !ok || !got.In(field.Inset(1)) {
					t.Fatal("label clipped at viewport edge or entered HUD")
				}
				for _, other := range placed {
					if got.Overlaps(other) {
						t.Fatal("neighboring hit/miss labels overlap")
					}
				}
				placed = append(placed, got)
			}
		}
	}
}

func TestCombatMarkersExpireAndKeepIndependentLanes(t *testing.T) {
	r := &Renderer{}
	s := domain.NewSessionSeed(21)
	e := domain.CombatEvent{Target: domain.Point{X: 12, Y: 10}, Level: 1, Damage: 20}
	for i := 0; i < 10; i++ {
		e.Damage = i
		r.ShowCombat(s, []domain.CombatEvent{e})
	}
	if len(r.combat.active) != combatLanes {
		t.Fatal("too many overlapping labels on one target")
	}
	used := map[int]bool{}
	for _, m := range r.combat.active {
		if used[m.lane] || m.event.Damage < 6 {
			t.Fatal("lane collision or old event retained")
		}
		used[m.lane] = true
	}
	for i := 0; i < combatMarkerTicks-1; i++ {
		r.Tick()
	}
	if len(r.combat.active) != combatLanes {
		t.Fatal("markers expired too soon")
	}
	r.Tick()
	if len(r.combat.active) != 0 {
		t.Fatal("markers did not expire without another turn")
	}
}

func TestCombatMarkersResetOnNewLevelOrSession(t *testing.T) {
	r := &Renderer{}
	s := domain.NewSessionSeed(21)
	e := domain.CombatEvent{Level: 1, Damage: 20}
	r.ShowCombat(s, []domain.CombatEvent{e})
	s.UpdateLevel()
	r.ShowCombat(s, []domain.CombatEvent{e})
	if len(r.combat.active) != 0 {
		t.Fatal("old floor effect leaked through portal")
	}
	e.Level = 2
	r.ShowCombat(s, []domain.CombatEvent{e})
	r.combat.sync(domain.NewSessionSeed(21))
	if len(r.combat.active) != 0 {
		t.Fatal("previous run effects survived restart")
	}
}

func TestCombatMarkerQueueIsBoundedAndCopiesEvents(t *testing.T) {
	r := &Renderer{}
	s := domain.NewSessionSeed(21)
	events := []domain.CombatEvent{{Level: 1, Damage: 7}}
	for i := 0; i < 100; i++ {
		events[0].Target.X = i
		r.ShowCombat(s, events)
	}
	events[0].Damage = 999
	if len(r.combat.active) != maxCombatMarkers || r.combat.active[0].event.Damage != 7 {
		t.Fatal("queue unbounded or aliases the reusable domain buffer")
	}
}

func TestCombatLabelsSeparateMissZeroAndMaxHP(t *testing.T) {
	for _, tc := range []struct {
		event domain.CombatEvent
		want  string
	}{
		{domain.CombatEvent{Damage: domain.Miss}, "MISS"},
		{domain.CombatEvent{Parried: true, TargetPlayer: true}, "PARRY"},
		{domain.CombatEvent{Damage: 54, Critical: true}, "-54!"},
		{domain.CombatEvent{Damage: 0}, "0"},
		{domain.CombatEvent{Damage: 25}, "-25"},
		{domain.CombatEvent{Damage: 30, MaxHP: true}, "-30 MAX HP"},
	} {
		if got, _ := combatLabel(tc.event); got != tc.want {
			t.Fatalf("label %q, want %q", got, tc.want)
		}
	}
	if label, ink := combatLabel(domain.CombatEvent{Damage: 25, TargetPlayer: true, Shielded: true}); label != "-25" || ink != color.NRGBA(uiShieldLight) {
		t.Fatalf("a hit the shield took whole: %q in %v, want grey", label, ink)
	}
	_, outgoing := combatLabel(domain.CombatEvent{Damage: 10})
	_, incoming := combatLabel(domain.CombatEvent{Damage: 10, TargetPlayer: true})
	if outgoing == incoming {
		t.Fatal("incoming damage should have a distinct color")
	}
}

func TestCriticalMarkerIsRed(t *testing.T) {
	_, crit := combatLabel(domain.CombatEvent{Damage: 54, Critical: true})
	_, hit := combatLabel(domain.CombatEvent{Damage: 54})
	if crit.R < 200 || crit.G > 80 || crit.B > 80 || crit == hit {
		t.Fatalf("critical marker must be red and differ from an ordinary hit: %+v", crit)
	}
}
