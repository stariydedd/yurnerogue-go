package domain

import "errors"

// RulesVersion must change when simulation rules or random call order changes.
const RulesVersion = "2"
const MaxReplayBytes = 60000
const MaxReplayTurns = 100000

var ErrAction = errors.New("invalid replay action")
var ErrReplayLimit = errors.New("replay limit exceeded")

// UseChoice shares inventory rules between the UI and the headless verifier.
// Weapon choice 0 returns to Quelling Blade; other categories use zero-based indices.
func (s *Session) UseChoice(t ItemType, choice int) bool {
	p := s.Player
	items := p.ItemsOfType(t)
	if t < ItemFood || t > ItemWeapon || choice < 0 {
		return false
	}
	rows := len(items)
	if t == ItemWeapon {
		if len(items) == 0 && p.Weapon == nil {
			return false
		}
		rows++
	}
	if choice >= rows {
		return false
	}
	if t == ItemWeapon {
		if choice == 0 {
			weapon := p.UnequipWeapon()
			if weapon == nil {
				return true
			}
			if p.PickUpItem(weapon) {
				s.SetMessage("You holstered " + weapon.Name + ".")
			} else {
				p.Weapon = weapon
				s.SetMessage("Backpack full! Cannot holster weapon.")
			}
			return true
		}
		item := items[choice-1]
		old := p.EquipWeapon(item)
		msg := "You equipped " + item.Name + "."
		if old != nil {
			if s.DropItemNearPlayer(old) {
				msg += " Dropped " + old.Name + "."
			} else {
				// EquipWeapon освободил слот оружия, поэтому прежнее помещается
				// даже в рюкзак, который был полон до смены.
				p.Backpack = append(p.Backpack, old)
				msg += " Stowed " + old.Name + " in backpack."
			}
		}
		s.SetMessage(msg)
		return true
	}

	item := items[choice]
	if !p.UseItem(item) {
		return true
	}
	s.SetMessage("You used " + item.Name + item.StatLabel() + ".")
	switch t {
	case ItemFood:
		s.Stats.FoodUsed++
	case ItemElixir:
		s.Stats.ElixirsUsed++
	case ItemScroll:
		s.Stats.ScrollsRead++
	}
	return true
}

// ApplyAction is the sole ranked input path. Menus/rendering consume no RNG.
// Lowercase WASD is a step; uppercase is a run; hjke + digit selects an item.
// t + lowercase direction is a Critical Strike; b is a Parry.
// z skips a turn: while asleep, or awake as a wait that lets enemies act.
func (s *Session) ApplyAction(action string) error {
	s.CombatEvents = s.CombatEvents[:0]
	if !s.Player.IsAlive() || s.Won() {
		return ErrAction
	}
	var dx, dy int
	run := false
	itemType := ItemNone
	choice := 0
	special := len(action) == 2 && action[0] == 't'
	guard := action == "b"
	parsed := action
	if special {
		parsed = action[1:]
	}
	switch parsed {
	case "w", "W":
		dy = -1
	case "s", "S":
		dy = 1
	case "a", "A":
		dx = -1
	case "d", "D":
		dx = 1
	case "z":
	case "b":
	default:
		if len(action) != 2 || action[1] < '0' || action[1] > '9' {
			return ErrAction
		}
		switch action[0] {
		case 'h':
			itemType = ItemWeapon
		case 'j':
			itemType = ItemFood
		case 'k':
			itemType = ItemElixir
		case 'e':
			itemType = ItemScroll
		default:
			return ErrAction
		}
		choice = int(action[1] - '0')
	}
	if special && (len(parsed) != 1 || !((parsed == "w") || (parsed == "a") || (parsed == "s") || (parsed == "d"))) {
		return ErrAction
	}
	if len(action) == 1 {
		run = action[0] >= 'A' && action[0] <= 'Z'
	}
	s.Message = ""
	if s.Player.Sleeping {
		s.SetMessage("You are asleep!")
		s.ResolveTurn()
	} else if special {
		if s.Player.StrikeCooldown > 0 {
			s.SetMessage(s.Player.StrikeChargeMessage())
			return ErrAction
		}
		op := s.OpponentAt(s.Player.X+dx, s.Player.Y+dy)
		if op == nil {
			s.SetMessage("No enemy in that direction.")
			return ErrAction
		}
		if dx != 0 {
			s.Player.Facing = dx
		}
		s.Player.powerStrike = true
		s.attack(op)
		s.Player.powerStrike = false
		s.Player.StrikeCooldown = StrikeRechargeAttacks
		s.ResolveTurn()
	} else if guard {
		if s.Player.GuardCooldown > 0 {
			s.SetMessage(s.Player.GuardChargeMessage())
			return ErrAction
		}
		// Parry is not a free wait: it needs an enemy that can hit now.
		if !s.enemyInContact() {
			s.SetMessage("No enemy next to you.")
			return ErrAction
		}
		s.Player.Guarding = true
		s.Player.GuardCooldown = GuardRechargeHits
		s.SetMessage("Parry: hits are blocked and struck back.")
		s.ResolveTurn()
	} else if itemType != ItemNone {
		if !s.UseChoice(itemType, choice) {
			return ErrAction
		}
		s.ResolveTurn()
	} else if parsed == "z" {
		s.SetMessage("You wait.")
		s.ResolveTurn()
	} else if run {
		s.Run(dx, dy)
	} else {
		acted := false
		if dx != 0 {
			acted = s.MoveX(dx)
		} else {
			acted = s.MoveY(dy)
		}
		if acted {
			s.ResolveTurn()
		}
	}
	if len(s.actions)+len(action) <= MaxReplayBytes && s.Turns <= MaxReplayTurns {
		s.actions = append(s.actions, action...)
	} else {
		s.ReplayOverflow = true
	}
	return nil
}

func (s *Session) Actions() string { return string(s.actions) }

// Replay recomputes a terminal run. No client-supplied statistics are used.
func Replay(seed int64, actions string) (*Session, error) {
	if len(actions) == 0 || len(actions) > MaxReplayBytes {
		return nil, ErrReplayLimit
	}
	s := NewSessionSeed(seed)
	for i := 0; i < len(actions); {
		n := 1
		switch actions[i] {
		case 'h', 'j', 'k', 'e', 't':
			n = 2
		}
		if i+n > len(actions) {
			return nil, ErrAction
		}
		if err := s.ApplyAction(actions[i : i+n]); err != nil {
			return nil, err
		}
		if s.ReplayOverflow {
			return nil, ErrReplayLimit
		}
		i += n
	}
	if s.Player.IsAlive() && !s.Won() {
		return nil, errors.New("run is not finished")
	}
	return s, nil
}
