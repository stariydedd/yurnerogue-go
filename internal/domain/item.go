package domain

import "math/rand"

// BaseWeaponName is the permanent starter, represented by no equipped upgrade.
const BaseWeaponName = "Quelling Blade"

// ItemType — категория предмета.
type ItemType int

const (
	ItemNone ItemType = iota
	ItemTreasure
	ItemFood
	ItemElixir
	ItemScroll
	ItemWeapon
)

// ItemSubType — какую характеристику меняет эликсир или свиток.
type ItemSubType int

const (
	SubNone ItemSubType = iota
	SubStrength
	SubAgility
	SubHealth
)

// Item — предмет: еда, эликсир, свиток, оружие или сокровище.
type Item struct {
	Type    ItemType
	SubType ItemSubType
	Name    string

	HealthEffect    int
	MaxHealthEffect int
	AgilityEffect   int
	StrengthEffect  int

	X, Y int
}

var (
	foodNames = []string{
		"Tango",
		"Iron Branch",
		"Faerie Fire",
		"Mango",
		"Healing Salve",
		"Lotus",
		"Cheese",
		"Seeds of Serenity",
		"Elixir",
	}
	elixirNames = []string{
		"Phantom Clarity",
		"Arcane Clarity",
		"Frozen Clarity",
		"Crimson Clarity",
		"Jade Clarity",
		"Moon Clarity",
		"Mystic Clarity",
		"Ember Clarity",
		"Wind Clarity",
	}
	scrollNames = []string{
		"Aghanim's Scroll",
		"Ogre Scroll",
		"Elven Scroll",
		"Vital Scroll",
		"Arcane Scroll",
		"Sage Scroll",
		"Power Scroll",
		"Mystic Scroll",
		"Ancient Scroll",
	}
	weaponNames = []string{
		"Yasha",
		"Diffusal Blade",
		"Butterfly",
		"Radiance",
		"Crystalys",
		"Battle Fury",
		"Desolator",
		"Shadow Blade",
		"Silver Edge",
		"Abyssal Blade",
	}
)

// pick возвращает случайный элемент списка.
func pick[T any](xs []T, rng ...*rand.Rand) T {
	return xs[random(source(rng)).Intn(len(xs))]
}

// rollUpTo возвращает случайное число от 1 до max (max не меньше 1).
func rollUpTo(max int, rng ...*rand.Rand) int {
	if max < 1 {
		max = 1
	}
	return random(source(rng)).Intn(max) + 1
}

// RandomItem создаёт случайный предмет: еду, эликсир, свиток или оружие.
// Величина эффектов считается от характеристик игрока, поэтому он нужен.
func RandomItem(player *Person) *Item {
	switch random(player.rng).Intn(4) {
	case 0:
		return NewFood(player)
	case 1:
		return NewElixir(player)
	case 2:
		return NewScroll(player)
	default:
		return NewWeapon(player.rng)
	}
}

// NewFood — еда, восстанавливающая до 20% максимального здоровья.
func NewFood(player *Person) *Item {
	return &Item{
		Type:         ItemFood,
		Name:         pick(foodNames, player.rng),
		HealthEffect: rollUpTo(player.MaxHealth*20/100, player.rng),
	}
}

// NewElixir — временный бафф к одной из характеристик на ElixirDuration ходов.
func NewElixir(player *Person) *Item {
	it := &Item{Type: ItemElixir, Name: pick(elixirNames, player.rng)}
	applyStatRoll(it, player)
	return it
}

// NewScroll — постоянный бафф к одной из характеристик.
func NewScroll(player *Person) *Item {
	it := &Item{Type: ItemScroll, Name: pick(scrollNames, player.rng)}
	applyStatRoll(it, player)
	return it
}

// applyStatRoll выбирает характеристику и величину бонуса: здоровье до 20%
// от максимума, ловкость и сила — до 10% от текущего значения.
func applyStatRoll(it *Item, player *Person) {
	switch random(player.rng).Intn(3) {
	case 0:
		it.SubType = SubHealth
		it.MaxHealthEffect = rollUpTo(player.MaxHealth*20/100, player.rng)
	case 1:
		it.SubType = SubAgility
		it.AgilityEffect = rollUpTo(player.Agility*10/100, player.rng)
	default:
		it.SubType = SubStrength
		it.StrengthEffect = rollUpTo(player.Strength*10/100, player.rng)
	}
}

// NewWeapon — оружие с бонусом к силе 30..50.
func NewWeapon(rng ...*rand.Rand) *Item {
	// Rules version 1 consumes Intn(9), then Intn(21). Keep those exact draws:
	// Intn(len(weaponNames)) would change damage and subsequent map/combat rolls.
	// Both existing rolls select the cosmetic name, allowing all ten variants.
	const nameRollBound = 9
	nameRoll := random(source(rng)).Intn(nameRollBound)
	damageRoll := random(source(rng)).Intn(21)
	return &Item{
		Type:           ItemWeapon,
		Name:           weaponNames[(nameRoll+damageRoll*nameRollBound)%len(weaponNames)],
		StrengthEffect: damageRoll + 30,
	}
}

// StatLabel — приписка к сообщению об использовании предмета, например " [+3 STR]".
func (it *Item) StatLabel() string {
	switch {
	case it.HealthEffect > 0:
		return " [+" + itoa(it.HealthEffect) + " HP]"
	case it.MaxHealthEffect > 0:
		return " [+" + itoa(it.MaxHealthEffect) + " MAX HP]"
	case it.AgilityEffect > 0:
		return " [+" + itoa(it.AgilityEffect) + " AGI]"
	case it.StrengthEffect > 0:
		return " [+" + itoa(it.StrengthEffect) + " STR]"
	}
	return ""
}
