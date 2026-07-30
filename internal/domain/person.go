package domain

// Person — персонаж игрока: характеристики, оружие, рюкзак, временные эффекты.
type Person struct {
	X, Y int

	MaxHealth int
	Health    int
	Agility   int
	Strength  int
	Weapon    *Item
	Treasures int

	Backpack []*Item

	// Facing: 1 — смотрит вправо, -1 — влево (последний горизонтальный шаг).
	Facing int

	// Sleeping и SleepTurns — эффект удара Skywrath Mage.
	Sleeping   bool
	SleepTurns int

	effects []statEffect
}

// statEffect — активный эффект эликсира, откатывается по истечении ходов.
type statEffect struct {
	sub       ItemSubType
	amount    int
	turnsLeft int
}

// NewPerson создаёт игрока с базовыми характеристиками вне карты.
func NewPerson() *Person {
	return &Person{
		X: -1, Y: -1,
		MaxHealth: DefaultMaxHealth,
		Health:    DefaultMaxHealth,
		Agility:   DefaultAgility,
		Strength:  DefaultStrength,
		Treasures: DefaultTreasures,
		Facing:    1,
	}
}

// IsAlive — жив ли персонаж.
func (p *Person) IsAlive() bool { return p.Health > 0 }

// TakeDamage наносит урон, здоровье не уходит ниже нуля.
func (p *Person) TakeDamage(damage int) {
	p.Health -= damage
	if p.Health < 0 {
		p.Health = 0
	}
}

// Heal лечит, не выше максимума.
func (p *Person) Heal(amount int) {
	p.Health += amount
	if p.Health > p.MaxHealth {
		p.Health = p.MaxHealth
	}
}

// IncreaseMaxHealth поднимает максимум здоровья и лечит на столько же.
func (p *Person) IncreaseMaxHealth(amount int) {
	p.MaxHealth += amount
	p.Heal(amount)
}

// FallAsleep усыпляет персонажа на turns ходов.
func (p *Person) FallAsleep(turns int) {
	p.Sleeping = true
	p.SleepTurns = turns
}

// TickSleep отсчитывает ход сна; возвращает true, пока персонаж спит.
func (p *Person) TickSleep() bool {
	if !p.Sleeping {
		return false
	}
	p.SleepTurns--
	if p.SleepTurns <= 0 {
		p.Sleeping = false
		p.SleepTurns = 0
	}
	return true
}

// PickUpItem кладёт предмет в рюкзак; false — превышен лимит на тип.
func (p *Person) PickUpItem(item *Item) bool {
	count := 0
	for _, it := range p.Backpack {
		if it.Type == item.Type {
			count++
		}
	}
	if count >= MaxBackpackItemsPerType {
		return false
	}
	p.Backpack = append(p.Backpack, item)
	return true
}

// ItemsOfType возвращает предметы рюкзака заданной категории.
func (p *Person) ItemsOfType(t ItemType) []*Item {
	var out []*Item
	for _, it := range p.Backpack {
		if it.Type == t {
			out = append(out, it)
		}
	}
	return out
}

// removeAt убирает предмет из рюкзака по индексу.
func (p *Person) removeAt(index int) bool {
	if index < 0 || index >= len(p.Backpack) {
		return false
	}
	p.Backpack = append(p.Backpack[:index], p.Backpack[index+1:]...)
	return true
}

// indexOf возвращает позицию предмета в рюкзаке или -1.
func (p *Person) indexOf(item *Item) int {
	for i, it := range p.Backpack {
		if it == item {
			return i
		}
	}
	return -1
}

// UseItem применяет предмет из рюкзака и убирает его оттуда.
// Эликсир действует ElixirDuration ходов, остальное — навсегда.
func (p *Person) UseItem(item *Item) bool {
	i := p.indexOf(item)
	if i < 0 {
		return false
	}
	if item.Type == ItemElixir {
		p.applyElixir(item)
	} else {
		p.Heal(item.HealthEffect)
		p.IncreaseMaxHealth(item.MaxHealthEffect)
		p.Agility += item.AgilityEffect
		p.Strength += item.StrengthEffect
	}
	p.removeAt(i)
	return true
}

// applyElixir выдаёт бонус и ставит его в очередь на откат.
func (p *Person) applyElixir(item *Item) {
	add := func(sub ItemSubType, amount int) {
		if amount == 0 {
			return
		}
		switch sub {
		case SubHealth:
			p.IncreaseMaxHealth(amount)
		case SubAgility:
			p.Agility += amount
		case SubStrength:
			p.Strength += amount
		}
		p.effects = append(p.effects, statEffect{sub: sub, amount: amount, turnsLeft: ElixirDuration})
	}
	add(SubHealth, item.MaxHealthEffect)
	add(SubAgility, item.AgilityEffect)
	add(SubStrength, item.StrengthEffect)
}

// TickEffects уменьшает таймеры эликсиров и откатывает истёкшие.
func (p *Person) TickEffects() {
	remaining := p.effects[:0]
	for _, e := range p.effects {
		e.turnsLeft--
		if e.turnsLeft > 0 {
			remaining = append(remaining, e)
			continue
		}
		switch e.sub {
		case SubHealth:
			p.MaxHealth -= e.amount
			if p.MaxHealth < 1 {
				p.MaxHealth = 1
			}
			if p.Health > p.MaxHealth {
				p.Health = p.MaxHealth
			}
		case SubAgility:
			p.Agility -= e.amount
		case SubStrength:
			p.Strength -= e.amount
		}
	}
	p.effects = remaining
}

// ActiveEffects — сколько эффектов эликсиров действует сейчас.
func (p *Person) ActiveEffects() int { return len(p.effects) }

// EquipWeapon экипирует оружие из рюкзака и возвращает прежнее (упадёт на пол).
func (p *Person) EquipWeapon(item *Item) *Item {
	i := p.indexOf(item)
	if i < 0 {
		return nil
	}
	p.removeAt(i)
	old := p.Weapon
	p.Weapon = item
	return old
}

// UnequipWeapon снимает оружие и возвращает его (в рюкзак не кладёт).
func (p *Person) UnequipWeapon() *Item {
	w := p.Weapon
	p.Weapon = nil
	return w
}

// AttackStrength — сила с учётом экипированного оружия.
func (p *Person) AttackStrength() int {
	if p.Weapon == nil {
		return p.Strength
	}
	return p.Strength + p.Weapon.StrengthEffect
}

// ReceiveTreasure начисляет сокровища за убитого врага.
func (p *Person) ReceiveTreasure(amount int) { p.Treasures += amount }
