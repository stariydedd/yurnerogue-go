package game

import (
	"reflect"
	"testing"

	"github.com/stariydedd/yurnerogue-go/internal/domain"
	"github.com/stariydedd/yurnerogue-go/internal/render"
	"github.com/stariydedd/yurnerogue-go/internal/sound"
)

func TestActionSoundsFollowSuccessfulGameEvents(t *testing.T) {
	before := actionAudioSnapshot{level: 1, items: 2, enemyHealth: 100}
	after := before
	after.stats = domain.Stats{AttacksMade: 1, HitsTaken: 1, EnemiesKilled: 1}
	after.enemyHealth = 0
	if got := actionCues(before, after, "d"); !reflect.DeepEqual(got, []sound.Cue{sound.Hit, sound.Hurt, sound.Kill}) {
		t.Fatalf("wrong combat cues %v", got)
	}
	if got := actionCues(before, before, "w"); len(got) != 0 {
		t.Fatal("blocked movement makes action sounds")
	}
	after = before
	after.stats.AttacksMade = 1
	if got := actionCues(before, after, "d"); !reflect.DeepEqual(got, []sound.Cue{sound.Swing}) {
		t.Fatalf("miss produced an impact: %v", got)
	}
	after = before
	after.stats.FoodUsed = 1
	after.items--
	if got := actionCues(before, after, "j0"); !reflect.DeepEqual(got, []sound.Cue{sound.Heal}) {
		t.Fatal("using food makes pickup sound")
	}
	after = before
	after.items++
	if got := actionCues(before, after, "d"); !reflect.DeepEqual(got, []sound.Cue{sound.Pickup}) {
		t.Fatal("pickup missing")
	}
	after = before
	after.level++
	if got := actionCues(before, after, "d"); !reflect.DeepEqual(got, []sound.Cue{sound.Portal}) {
		t.Fatal("portal missing")
	}
}

func TestSoundObservationDoesNotChangeReplay(t *testing.T) {
	g := &Game{session: domain.NewSessionSeed(21), state: StatePlaying, renderer: &render.Renderer{}}
	twin := domain.NewSessionSeed(21)
	for _, action := range []string{"s", "s", "d", "a", "w", "h0"} {
		g.performAction(action)
		_ = twin.ApplyAction(action)
	}
	if g.session.Actions() != twin.Actions() || !reflect.DeepEqual(g.session.Stats, twin.Stats) || !reflect.DeepEqual(g.session.Player, twin.Player) {
		t.Fatal("sound observation altered game simulation")
	}
}

func TestZeroDamageHitSoundMatchesCombatMarker(t *testing.T) {
	before := actionAudioSnapshot{level: 1, enemyHealth: 100}
	after := before
	after.stats.AttacksMade = 1
	after.combat = []domain.CombatEvent{{Damage: 0, Level: 1}}
	if got := actionCues(before, after, "d"); !reflect.DeepEqual(got, []sound.Cue{sound.Hit}) {
		t.Fatalf("successful zero-damage hit played a miss: %v", got)
	}
}
