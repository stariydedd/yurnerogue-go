package game

import (
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/stariydedd/yurnerogue-go/internal/domain"
	"github.com/stariydedd/yurnerogue-go/internal/sound"
)

func TestStrikeSelectionDoesNotSpendTurnAndRecordsDirection(t *testing.T) {
	s := domain.NewSessionSeed(21)
	p := s.Player
	p.X, p.Y, p.Agility = 12, 10, 1000000
	enemy := domain.NewOpponent(domain.Zombie)
	enemy.X, enemy.Y, enemy.Health = 13, 10, 1000
	s.Level.Rooms = []*domain.Room{{X: 8, Y: 6, W: 16, H: 9, Enemies: []*domain.Opponent{enemy}}}
	s.Level.Items = nil
	s.Level.Exit = domain.Point{X: 20, Y: 12}
	g := &Game{session: s, state: StatePlaying}
	g.HandleKey(ebiten.KeyF)
	if !p.StrikeArmed || s.Turns != 0 || s.Actions() != "" {
		t.Fatal("arming must be free")
	}
	g.HandleKey(ebiten.KeyEscape)
	if p.StrikeArmed || s.Turns != 0 {
		t.Fatal("cancel spent a turn")
	}
	g.HandleKey(ebiten.KeyF)
	g.HandleKey(ebiten.KeyRight)
	if p.StrikeArmed || s.Actions() != "td" || s.Turns != 1 || p.StrikeCooldown != domain.StrikeRechargeAttacks {
		t.Fatal("strike input did not enter shared replay path")
	}
	g.HandleKey(ebiten.KeyF)
	if p.StrikeArmed || s.Turns != 1 {
		t.Fatal("cooling strike armed")
	}
	g.HandleKey(ebiten.KeyE)
	if s.Actions() != "tdb" || p.GuardCooldown != domain.GuardRechargeHits || s.Turns != 2 {
		t.Fatal("guard input did not enter shared replay path")
	}
}

func TestStrikeSelectionClearsWhenOpeningPause(t *testing.T) {
	g := &Game{session: domain.NewSessionSeed(21), state: StatePlaying}
	g.HandleKey(ebiten.KeyF)
	g.HandleKey(ebiten.KeyQ)
	if g.session.Player.StrikeArmed || g.state != StatePauseMenu {
		t.Fatal("armed strike leaked into pause")
	}
}

func TestItemKeysOpenFoodAndClarityMenus(t *testing.T) {
	for key, want := range map[ebiten.Key]domain.ItemType{ebiten.KeyC: domain.ItemFood, ebiten.KeyX: domain.ItemElixir} {
		g := &Game{session: domain.NewSessionSeed(21), state: StatePlaying}
		p := g.session.Player
		p.Backpack = append(p.Backpack, domain.NewFood(p, 1), domain.NewElixir(p, 1))
		g.HandleKey(key)
		if g.state != StateItemMenu || g.itemMenuType != want {
			t.Fatalf("key %v did not open its item menu", key)
		}
	}
	g := &Game{session: domain.NewSessionSeed(21), state: StatePlaying}
	g.session.Player.Backpack = append(g.session.Player.Backpack, domain.NewFood(g.session.Player, 1))
	for _, old := range []ebiten.Key{ebiten.KeyJ, ebiten.KeyK, ebiten.KeyH} {
		g.HandleKey(old)
		if g.state != StatePlaying || g.session.Player.StrikeArmed || g.session.Actions() != "" {
			t.Fatalf("old binding %v still acts", old)
		}
	}
}

func TestZKeyWaitsATurn(t *testing.T) {
	g := &Game{session: domain.NewSessionSeed(21), state: StatePlaying}
	g.HandleKey(ebiten.KeyZ)
	if g.session.Actions() != "z" || g.session.Turns != 1 || g.state != StatePlaying {
		t.Fatalf("Z did not wait: actions %q turns %d", g.session.Actions(), g.session.Turns)
	}
}

func TestWeaponSoundFollowsTheLog(t *testing.T) {
	has := func(cues []sound.Cue, want sound.Cue) bool {
		for _, c := range cues {
			if c == want {
				return true
			}
		}
		return false
	}
	yasha := actionAudioSnapshot{weaponName: "Yasha", weaponBonus: 7}
	for _, tc := range []struct {
		after          actionAudioSnapshot
		sharpen, equip bool
	}{
		{actionAudioSnapshot{weaponName: "Yasha", weaponBonus: 8}, true, false},      // sharpened
		{actionAudioSnapshot{weaponName: "Yasha", weaponBonus: 9}, true, false},      // stronger copy, same name
		{actionAudioSnapshot{weaponName: "Butterfly", weaponBonus: 38}, false, true}, // new name
		{actionAudioSnapshot{weaponName: "Yasha", weaponBonus: 7}, false, false},     // nothing changed
	} {
		cues := actionCues(yasha, tc.after, "d")
		if has(cues, sound.Sharpen) != tc.sharpen || has(cues, sound.Equip) != tc.equip {
			t.Fatalf("%+v: cues %v", tc.after, cues)
		}
	}
}

func TestParriedHitPlaysClash(t *testing.T) {
	blocked := actionAudioSnapshot{combat: []domain.CombatEvent{
		{TargetPlayer: true, Parried: true},
		{TargetPlayer: true, Parried: true},
		{Damage: 18},
	}}
	clashes := 0
	for _, c := range actionCues(actionAudioSnapshot{}, blocked, "b") {
		if c == sound.Parry {
			clashes++
		}
	}
	if clashes != 1 {
		t.Fatalf("two parried hits must clang once, got %d", clashes)
	}
	for _, c := range actionCues(actionAudioSnapshot{}, actionAudioSnapshot{combat: []domain.CombatEvent{{TargetPlayer: true, Damage: 20}}}, "d") {
		if c == sound.Parry {
			t.Fatal("an ordinary hit must not clang")
		}
	}
}

func TestCriticalStrikeHitPlaysBladeDance(t *testing.T) {
	has := func(cues []sound.Cue, want sound.Cue) bool {
		for _, c := range cues {
			if c == want {
				return true
			}
		}
		return false
	}
	before := actionAudioSnapshot{enemyHealth: 100}
	hit := actionAudioSnapshot{enemyHealth: 40, stats: domain.Stats{AttacksMade: 1}, combat: []domain.CombatEvent{{Damage: 60}}}
	miss := actionAudioSnapshot{enemyHealth: 100, stats: domain.Stats{AttacksMade: 1}, combat: []domain.CombatEvent{{Damage: domain.Miss}}}
	if cues := actionCues(before, hit, "td"); !has(cues, sound.Critical) || has(cues, sound.Hit) {
		t.Fatalf("critical hit cues: %v", cues)
	}
	if cues := actionCues(before, hit, "d"); has(cues, sound.Critical) || !has(cues, sound.Hit) {
		t.Fatalf("ordinary hit cues: %v", cues)
	}
	if cues := actionCues(before, miss, "td"); has(cues, sound.Critical) || !has(cues, sound.Swing) {
		t.Fatalf("critical miss cues: %v", cues)
	}
}
