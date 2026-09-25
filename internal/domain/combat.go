package domain

import "math/rand"

// Miss — результат атаки при промахе (в отличие от нуля урона у отдыхающего Axe).
const Miss = -1

// checkHit — попадание зависит от разницы в ловкости атакующего и цели.
func checkHit(attackerAgility, defenderAgility int, rng ...*rand.Rand) bool {
	chance := InitialHitChance + float64(attackerAgility-defenderAgility-StandardAgility)*AgilityFactor
	return random(source(rng)).Intn(100) < clamp(int(chance), 0, 100)
}

// calculateLoot — сокровища, выпадающие из поверженного врага.
func calculateLoot(o *Opponent, rng ...*rand.Rand) int {
	return int(float64(o.Agility)*0.2+float64(o.Health)*0.5+float64(o.Strength)*0.5) + random(source(rng)).Intn(20)
}

// WeaponDamage: урон одного удара оружием при текущей силе; nil означает
// стартовый Quelling Blade. Бонус оружия прибавляется к урону клинка и растёт
// вместе с силой: при стандартной силе каждый пункт бонуса даёт 1 урон.
func WeaponDamage(p *Person, weapon *Item) int {
	damage := InitialDamage + int(float64(p.Strength-StandardStrength)*StrengthFactor)
	if weapon != nil {
		damage += weapon.StrengthEffect * (p.Strength + StrengthAddition) / (StandardStrength + StrengthAddition)
	}
	return max(0, damage)
}

// PlayerAttacks — атака игрока по врагу. Возвращает урон или Miss.
func PlayerAttacks(p *Person, o *Opponent) int {
	// Bloodseeker отражает первый удар по себе.
	if o.DeflectsFirstStrike() {
		return Miss
	}
	if !checkHit(p.Agility, o.Agility, p.rng) {
		return Miss
	}

	damage := WeaponDamage(p, p.Weapon)
	if p.powerStrike {
		damage = (damage*3 + 1) / 2
	}

	o.TakeDamage(damage)
	if !o.IsAlive() {
		p.ReceiveTreasure(calculateLoot(o, p.rng))
	}
	return damage
}

// OpponentAttacks — атака врага по игроку. Возвращает урон, Miss при промахе
// или 0, когда Axe отдыхает после удара.
func OpponentAttacks(o *Opponent, p *Person) int {
	// Удар Axe нельзя увернуться.
	if o.Type != Ogre && !checkHit(o.Agility, p.Agility, p.rng) {
		return Miss
	}

	// Bloodseeker крадёт максимум здоровья вместо обычного урона.
	if o.Type == Vampire {
		if p.Guarding {
			return 0 // parried: nothing is drained
		}
		damage := o.MaxHealthDrain()
		p.DrainMaxHealth(damage)
		return damage
	}

	var damage int
	if o.Type == Ogre {
		if o.Resting() {
			o.SetResting(false)
			return 0
		}
		o.SetResting(true)
		damage = int(float64(o.Strength-StandardStrength) * StrengthFactor)
	} else {
		damage = InitialDamage + int(float64(o.Strength-StandardStrength)*StrengthFactor)
	}
	if damage < 0 {
		damage = 0
	}
	// A parried hit deals no damage and cannot put the hero to sleep.
	if p.Guarding {
		return 0
	}
	p.TakeDamage(damage)

	// Skywrath Mage может усыпить игрока на ход.
	if o.Type == Snake && random(p.rng).Intn(100) < SleepChance {
		p.FallAsleep(1)
	}
	return damage
}

// riposte: ответный удар парирования по врагу, чей удар игрок только что отпарировал.
// Он не отражается Bloodseeker и не считается атакой игрока в статистике.
func (s *Session) riposte(o *Opponent) {
	p := s.Player
	damage := WeaponDamage(p, p.Weapon) * RiposteDamagePercent / 100
	o.TakeDamage(damage)
	s.recordCombat(CombatEvent{Target: Point{X: o.X, Y: o.Y}, Damage: damage})
	name := o.Type.DisplayName()
	if o.IsAlive() {
		s.SetMessage("You parried the " + name + " and struck back for " + itoa(damage) + " dmg.")
		return
	}
	gold := calculateLoot(o, p.rng)
	p.ReceiveTreasure(gold)
	s.Stats.EnemiesKilled++
	s.SetMessage("You parried and killed the " + name + " for " + itoa(damage) + " dmg! Gained " + itoa(gold) + " gold.")
}

// enemyInContact: может ли хоть один живой враг ударить игрока прямо сейчас.
func (s *Session) enemyInContact() bool {
	for _, op := range s.Level.AllOpponents() {
		if op.IsAlive() && inContact(op, s.Player) {
			return true
		}
	}
	return false
}

// inContact — стоит ли враг вплотную к игроку (Skywrath достаёт и по диагонали).
func inContact(o *Opponent, p *Person) bool {
	dx, dy := abs(o.X-p.X), abs(o.Y-p.Y)
	if dx+dy <= 1 {
		return true
	}
	return o.Type == Snake && dx == 1 && dy == 1
}

// ProcessEnemyTurns — ход всех живых врагов: атака при контакте либо движение.
// Здесь же тикают сон игрока и эффекты эликсиров.
func (s *Session) ProcessEnemyTurns() {
	p := s.Player
	// Отсчитываем уже пропущенный ход до атак: новый сон должен сохраниться
	// до следующего действия игрока, в том числе при повторном усыплении.
	p.TickSleep()
	var living []*Opponent
	for _, o := range s.Level.AllOpponents() {
		if o.IsAlive() {
			living = append(living, o)
		}
	}

	for _, o := range living {
		if !inContact(o, p) {
			o.Move(p.X, p.Y, s.Level.Rooms, s.Level.Passages, living)
			continue
		}
		o.FacePlayer(p.X)
		if o.Type == Ghost {
			o.IsChasing = true // Riki перестаёт прятаться, когда бьёт
		}
		resting := o.Type == Ogre && o.Resting()
		damage := OpponentAttacks(o, p)
		parried := p.Guarding && !resting && damage != Miss
		if !resting {
			s.recordCombat(CombatEvent{
				Target: Point{X: p.X, Y: p.Y}, Damage: damage,
				TargetPlayer: true, MaxHP: o.Type == Vampire && damage > 0, Parried: parried,
			})
		}
		if damage > 0 {
			s.Stats.HitsTaken++
		}
		// A landed hit charges the parry, except hits taken while parrying.
		if !resting && damage != Miss && !p.Guarding {
			p.GuardCooldown = max(0, p.GuardCooldown-1)
		}
		if parried {
			s.riposte(o) // its message replaces the enemy's zero-damage hit
			continue
		}
		s.SetMessage(attackMessage(o, p, damage))
	}

	p.TickEffects()

	// Riki виден только изредка, пока не преследует игрока.
	for _, o := range living {
		o.IsVisible = o.Type != Ghost || o.IsChasing || random(p.rng).Intn(100) < ChanceGhostVisible
	}
}

// attackMessage — строка в HUD по результату вражеской атаки.
func attackMessage(o *Opponent, p *Person, damage int) string {
	name := o.Type.DisplayName()
	switch {
	case damage == Miss:
		return "The " + name + " missed you."
	case damage == 0:
		return "The " + name + " is preparing to strike..."
	case o.Type == Vampire:
		return "The " + name + " drained your max HP by " + itoa(damage) + "!"
	}
	msg := "The " + name + " hit you for " + itoa(damage) + " dmg."
	if p.Sleeping {
		msg += " You fall asleep!"
	}
	return msg
}
