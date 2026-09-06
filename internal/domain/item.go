package domain

import "math/rand"

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
		"Ration of the Ironclad",
		"Crimson Berry Cluster",
		"Loaf of the Forgotten Baker",
		"Smoked Wyrm Jerky",
		"Golden Apple of Vitality",
		"Hardtack of the Endless March",
		"Spiced Venison Strips",
		"Honeyed Nectar Bread",
		"Dried Mushrooms of the Deep",
	}
	elixirNames = []string{
		"Elixir of the Jade Serpent",
		"Potion of the Phantom's Breath",
		"Vial of Crimson Vitality",
		"Draught of the Frozen Star",
		"Elixir of the Shattered Mind",
		"Potion of the Wandering Soul",
		"Vial of Ember Essence",
		"Elixir of the Obsidian Veil",
		"Potion of the Howling Wind",
	}
	scrollNames = []string{
		"Scroll of Shadowstep",
		"Parchment of Eternal Flame",
		"Manuscript of Forgotten Truths",
		"Scroll of Iron Will",
		"Vellum of the Void",
		"Scroll of Whispers",
		"Tome of the Lost King",
		"Scroll of Unseen Paths",
		"Parchment of Thunderous Roar",
	}
	weaponNames = []string{
		"Blade of the Forgotten Dawn",
		"Obsidian Reaver",
		"Fang of the Shadow Wolf",
		"Ironclad Cleaver",
		"Crimson Talon",
		"Thunderstrike Maul",
		"Serpent's Kiss Dagger",
		"Voidrend Sword",
		"Ebonheart Spear",
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
	return &Item{
		Type:           ItemWeapon,
		Name:           pick(weaponNames, rng...),
		StrengthEffect: random(source(rng)).Intn(21) + 30,
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
