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
		{actionAudioSnapshot{weaponName: "Yasha", weaponBonus: 8}, true, false},      // заточено
		{actionAudioSnapshot{weaponName: "Yasha", weaponBonus: 9}, true, false},      // более сильная копия, то же название
		{actionAudioSnapshot{weaponName: "Butterfly", weaponBonus: 38}, false, true}, // новое название
		{actionAudioSnapshot{weaponName: "Yasha", weaponBonus: 7}, false, false},     // ничего не изменилось
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

func pacingFixture() (*Game, *domain.Session, *domain.Opponent) {
	s := domain.NewSessionSeed(21)
	p := s.Player
	p.X, p.Y, p.Agility = 12, 10, 1000000
	enemy := domain.NewOpponent(domain.Zombie)
	enemy.X, enemy.Y, enemy.Health = 13, 10, 100000
	s.Level.Rooms = []*domain.Room{{X: 8, Y: 6, W: 16, H: 9, Enemies: []*domain.Opponent{enemy}}}
	s.Level.Items = nil
	s.Level.Exit = domain.Point{X: 20, Y: 12}
	return &Game{session: s, state: StatePlaying, paced: true}, s, enemy
}

func TestRapidAttacksKeepOneSwingPerStepAndQueueTheLatest(t *testing.T) {
	g, s, enemy := pacingFixture()
	p := s.Player
	for i := 0; i < 5; i++ { // пять нажатий за один кадр
		g.HandleKey(ebiten.KeyRight)
	}
	if s.Actions() != "d" || g.queuedAction != "d" || !g.queuedAttack {
		t.Fatalf("taps inside the interval must give one swing and one queued, got %q", s.Actions())
	}
	g.ticks = attackInterval - 1
	g.releaseQueuedAction()
	if s.Actions() != "d" {
		t.Fatal("queued swing came before the interval")
	}
	g.ticks = attackInterval
	g.releaseQueuedAction()
	if s.Actions() != "dd" || g.queuedAction != "" {
		t.Fatal("queued swing did not come right after the interval")
	}

	// Удар из очереди, чей враг ушёл, не превращается в шаг.
	enemy.X, enemy.Y = 13, 10
	g.HandleKey(ebiten.KeyRight)
	enemy.Health = 0
	g.ticks = g.turnReady
	g.releaseQueuedAction()
	if s.Actions() != "dd" || p.X != 12 {
		t.Fatal("a stale swing moved the hero")
	}
}

func TestRapidStepsFollowTheStepAnimation(t *testing.T) {
	g, s, enemy := pacingFixture()
	enemy.Health = 0
	for i := 0; i < 4; i++ { // частые нажатия: один шаг, последнее нажатие ждёт
		g.HandleKey(ebiten.KeyUp)
		g.HandleKey(ebiten.KeyLeft)
	}
	if s.Actions() != "w" || g.queuedAction != "a" {
		t.Fatalf("mashed steps must not outrun the animation, got %q queued %q", s.Actions(), g.queuedAction)
	}
	g.ticks = turnInterval
	g.releaseQueuedAction()
	if s.Actions() != "wa" {
		t.Fatal("the latest press did not step once the animation ended")
	}
	if turnInterval > attackInterval {
		t.Fatal("a swing must not be quicker than a step")
	}
}

// Предмет, выбранный в меню сразу после шага, ждёт этот шаг. Возврат
// в игру раньше стирал его, и еда так и не съедалась.
func TestItemPickedRightAfterAStepIsNotLost(t *testing.T) {
	g, s, enemy := pacingFixture()
	enemy.Health = 0
	s.Player.Health = 100
	s.Player.PickUpItem(domain.NewFood(s.Player, 1))
	frame := func(key ebiten.Key) {
		before := g.state
		g.HandleKey(key)
		g.settleStateChange(before)
	}
	frame(ebiten.KeyUp) // шаг: следующий ход ждёт его анимацию
	frame(ebiten.KeyC)  // меню еды
	frame(ebiten.Key1)  // съесть, ещё внутри интервала шага
	if g.state != StatePlaying || g.queuedAction != "j0" {
		t.Fatalf("the pick must wait in the queue, state %v queued %q", g.state, g.queuedAction)
	}
	g.ticks = g.turnReady
	g.releaseQueuedAction()
	if s.Actions() != "wj0" || s.Stats.FoodUsed != 1 {
		t.Fatalf("the food was not eaten: actions %q", s.Actions())
	}

	// Уход из игры по-прежнему сбрасывает ход из очереди.
	g.ticks = 0
	g.turnReady = turnInterval
	frame(ebiten.KeyUp)
	frame(ebiten.KeyQ) // пауза
	if g.state != StatePauseMenu || g.queuedAction != "" {
		t.Fatalf("a turn queued before the pause survived it: %q", g.queuedAction)
	}
}
