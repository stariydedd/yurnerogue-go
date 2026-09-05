package domain

import "math/rand"

// Miss — результат атаки при промахе (в отличие от нуля урона у отдыхающего Axe).
const Miss = -1

// checkHit — попадание зависит от разницы в ловкости атакующего и цели.
func checkHit(attackerAgility, defenderAgility int) bool {
	chance := InitialHitChance + float64(attackerAgility-defenderAgility-StandardAgility)*AgilityFactor
	return rand.Intn(100) < clamp(int(chance), 0, 100)
}

// calculateLoot — сокровища, выпадающие из поверженного врага.
func calculateLoot(o *Opponent) int {
	return int(float64(o.Agility)*0.2+float64(o.Health)*0.5+float64(o.Strength)*0.5) + rand.Intn(20)
}

// PlayerAttacks — атака игрока по врагу. Возвращает урон или Miss.
func PlayerAttacks(p *Person, o *Opponent) int {
	// Bloodseeker отражает первый удар по себе.
	if o.DeflectsFirstStrike() {
		return Miss
	}
	if !checkHit(p.Agility, o.Agility) {
		return Miss
	}

	var damage int
	if p.Weapon != nil {
		damage = p.Weapon.StrengthEffect * (p.Strength + StrengthAddition) / 100
	} else {
		damage = InitialDamage + int(float64(p.Strength-StandardStrength)*StrengthFactor)
	}
	if damage < 0 {
		damage = 0
	}

	o.TakeDamage(damage)
	if !o.IsAlive() {
		p.ReceiveTreasure(calculateLoot(o))
	}
	return damage
}

// OpponentAttacks — атака врага по игроку. Возвращает урон, Miss при промахе
// или 0, когда Axe отдыхает после удара.
func OpponentAttacks(o *Opponent, p *Person) int {
	// Удар Axe нельзя увернуться.
	if o.Type != Ogre && !checkHit(o.Agility, p.Agility) {
		return Miss
	}

	// Bloodseeker крадёт максимум здоровья вместо обычного урона.
	if o.Type == Vampire {
		damage := max(1, p.MaxHealth/MaxHPPart)
		p.MaxHealth -= damage
		if p.Health > p.MaxHealth {
			p.Health = p.MaxHealth
		}
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
	p.TakeDamage(damage)

	// Skywrath Mage может усыпить игрока на ход.
	if o.Type == Snake && rand.Intn(100) < SleepChance {
		p.FallAsleep(1)
	}
	return damage
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
		damage := OpponentAttacks(o, p)
		if damage > 0 {
			s.Stats.HitsTaken++
		}
		s.SetMessage(attackMessage(o, p, damage))
	}

	p.TickEffects()

	// Riki виден только изредка, пока не преследует игрока.
	for _, o := range living {
		o.IsVisible = o.Type != Ghost || o.IsChasing || rand.Intn(100) < ChanceGhostVisible
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
