package domain

import "testing"

// sharpPlayer — игрок с запредельными характеристиками: попадание и убийство
// с одного удара гарантированы, тест не зависит от бросков.
func sharpPlayer() *Person {
	p := NewPerson()
	p.Agility = 1_000_000
	p.Strength = 1_000_000
	return p
}

func TestVampireDeflectsFirstStrike(t *testing.T) {
	p := sharpPlayer()
	v := NewOpponent(Vampire)

	if got := PlayerAttacks(p, v); got != Miss {
		t.Fatalf("первый удар по Bloodseeker должен отражаться, получено %d", got)
	}
	if got := PlayerAttacks(p, v); got <= 0 {
		t.Fatalf("второй удар должен проходить, получено %d", got)
	}
}

func TestVampireDrainsMaxHealth(t *testing.T) {
	p := NewPerson()
	v := NewOpponent(Vampire)
	v.Agility = 1_000_000 // гарантируем попадание
	before := p.MaxHealth

	damage := OpponentAttacks(v, p)

	if damage <= 0 {
		t.Fatalf("урон %d, ожидался положительный", damage)
	}
	if p.MaxHealth >= before {
		t.Fatalf("максимум здоровья не уменьшился: было %d, стало %d", before, p.MaxHealth)
	}
}

func TestKillingOpponentGrantsTreasure(t *testing.T) {
	p := sharpPlayer()
	// Ненулевые характеристики: лут считается от них, иначе бросок может дать 0.
	op := NewOpponent(Zombie)
	op.Health = 1

	damage := PlayerAttacks(p, op)

	if damage <= 0 {
		t.Fatalf("урон %d, ожидался положительный", damage)
	}
	if op.IsAlive() {
		t.Fatal("враг с 1 HP должен умереть от удара")
	}
	if p.Treasures <= 0 {
		t.Fatalf("сокровищ %d, ожидались положительные", p.Treasures)
	}
}

func TestOgreRestsAfterAttack(t *testing.T) {
	// Axe бьёт, затем ход отдыхает (нулевой урон), затем снова бьёт.
	p := NewPerson()
	o := NewOpponent(Ogre)

	if first := OpponentAttacks(o, p); first <= 0 {
		t.Fatalf("первый удар Axe должен нанести урон, получено %d", first)
	}
	if rest := OpponentAttacks(o, p); rest != 0 {
		t.Fatalf("следующим ходом Axe отдыхает, ожидался 0, получено %d", rest)
	}
	if third := OpponentAttacks(o, p); third <= 0 {
		t.Fatalf("после отдыха Axe снова бьёт, получено %d", third)
	}
}

func TestElixirEffectExpires(t *testing.T) {
	p := NewPerson()
	elixir := &Item{Type: ItemElixir, SubType: SubStrength, StrengthEffect: 25}
	base := p.Strength
	p.Backpack = append(p.Backpack, elixir)

	if !p.UseItem(elixir) {
		t.Fatal("эликсир из рюкзака должен использоваться")
	}
	if p.Strength != base+25 {
		t.Fatalf("сила %d, ожидалась %d", p.Strength, base+25)
	}
	for i := 0; i < ElixirDuration; i++ {
		p.TickEffects()
	}
	if p.Strength != base {
		t.Fatalf("после истечения сила %d, ожидался откат к %d", p.Strength, base)
	}
	if p.ActiveEffects() != 0 {
		t.Fatalf("осталось %d активных эффектов", p.ActiveEffects())
	}
}

func TestScrollBuffIsPermanent(t *testing.T) {
	p := NewPerson()
	scroll := &Item{Type: ItemScroll, SubType: SubStrength, StrengthEffect: 10}
	base := p.Strength
	p.Backpack = append(p.Backpack, scroll)

	p.UseItem(scroll)
	for i := 0; i < ElixirDuration*2; i++ {
		p.TickEffects()
	}
	if p.Strength != base+10 {
		t.Fatalf("бафф свитка должен быть постоянным: %d, ожидалось %d", p.Strength, base+10)
	}
}

func TestBackpackLimitPerType(t *testing.T) {
	p := NewPerson()
	for i := 0; i < MaxBackpackItemsPerType; i++ {
		if !p.PickUpItem(NewFood(p)) {
			t.Fatalf("предмет %d должен помещаться в рюкзак", i+1)
		}
	}
	if p.PickUpItem(NewFood(p)) {
		t.Fatal("сверх лимита предмет того же типа браться не должен")
	}
	if !p.PickUpItem(NewWeapon()) {
		t.Fatal("лимит считается по типам: оружие должно помещаться")
	}
}

func TestEquipWeaponReturnsOldOne(t *testing.T) {
	p := NewPerson()
	first, second := NewWeapon(), NewWeapon()
	p.Backpack = append(p.Backpack, first, second)

	if old := p.EquipWeapon(first); old != nil {
		t.Fatal("с голыми руками менять нечего")
	}
	old := p.EquipWeapon(second)
	if old != first {
		t.Fatal("экипировка должна вернуть прежнее оружие, чтобы оно упало на пол")
	}
	if p.Weapon != second {
		t.Fatal("экипировано не то оружие")
	}
	if p.AttackStrength() != p.Strength+second.StrengthEffect {
		t.Fatal("сила атаки должна учитывать оружие")
	}
}

func TestSleepTicksDown(t *testing.T) {
	p := NewPerson()
	p.FallAsleep(1)

	if !p.TickSleep() {
		t.Fatal("первый тик приходится на сон")
	}
	if p.Sleeping {
		t.Fatal("сон на один ход должен закончиться после тика")
	}
}

func TestEnemyTurnsAttackAdjacentPlayer(t *testing.T) {
	s := cleanSession(t)
	room := s.Level.RoomAt(s.Player.X, s.Player.Y)
	op := NewOpponent(Zombie)
	op.Agility = 1_000_000 // гарантируем попадание
	op.X, op.Y = s.Player.X+1, s.Player.Y
	room.Enemies = append(room.Enemies, op)
	hpBefore := s.Player.Health

	s.ProcessEnemyTurns()

	if s.Player.Health >= hpBefore {
		t.Fatalf("здоровье не убавилось: было %d, стало %d", hpBefore, s.Player.Health)
	}
	if s.Stats.HitsTaken != 1 {
		t.Fatalf("получено ударов %d, ожидался 1", s.Stats.HitsTaken)
	}
	if op.Facing != -1 {
		t.Fatal("враг должен повернуться к игроку (тот слева)")
	}
}
