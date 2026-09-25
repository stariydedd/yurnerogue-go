package game

import (
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/stariydedd/yurnerogue-go/internal/domain"
	"github.com/stariydedd/yurnerogue-go/internal/sound"
)

// Only the interactive entry point enables audio; replay verification stays silent.
func (g *Game) EnableAudio() {
	if g.audio == nil {
		g.audio = sound.New()
	}
}

func (g *Game) updateAudio(before State) {
	g.audio.SilenceMusic(g.state == StateWin)
	g.audio.Update(g.session != nil, ebiten.IsFocused())
	if before == g.state {
		return
	}
	switch g.state {
	case StateDeath:
		g.audio.Play(sound.Death)
	case StateWin:
		g.audio.Play(sound.Victory)
	case StatePlaying:
		if before == StateStarting {
			g.audio.Play(sound.Start)
		} else if before != StateItemMenu {
			g.audio.Play(sound.Click)
		}
	default:
		g.audio.Play(sound.Click)
	}
}

type actionAudioSnapshot struct {
	stats                     domain.Stats
	level, items, enemyHealth int
	weaponName                string
	weaponBonus               int
	position                  domain.Point
	stepCue                   sound.Cue
	combat                    []domain.CombatEvent
}

func captureActionAudio(s *domain.Session) actionAudioSnapshot {
	health := 0
	for _, enemy := range s.Opponents() {
		health += max(0, enemy.Health)
	}
	step := sound.StepGrass
	// Same geometry as the renderer's PathCells, independent of items/enemies
	// drawn over the floor and of which adjacent cells have been revealed.
	if domain.IsCorridorCell(s.Player.X, s.Player.Y, s.Level.Rooms, s.Level.Passages) {
		step = sound.StepTrail
	}
	bonus, name := 0, domain.BaseWeaponName
	if s.Player.Weapon != nil {
		bonus, name = s.Player.Weapon.StrengthEffect, s.Player.Weapon.Name
	}
	return actionAudioSnapshot{
		stats: s.Stats, level: s.LevelNum, items: len(s.Player.Backpack), enemyHealth: health,
		weaponName: name, weaponBonus: bonus, position: domain.Point{X: s.Player.X, Y: s.Player.Y}, stepCue: step,
		combat: append([]domain.CombatEvent(nil), s.CombatEvents...),
	}
}

func actionCues(before, after actionAudioSnapshot, action string) []sound.Cue {
	var cues []sound.Cue
	// RUN resolves multiple tiles instantly: emit one landing step, not a queued
	// trail of sounds after the player has stopped. Teleports aren't footsteps.
	if after.level == before.level && after.stats.TilesMoved > before.stats.TilesMoved && after.position != before.position {
		cues = append(cues, after.stepCue)
	}
	if after.level > before.level {
		cues = append(cues, sound.Portal)
	}
	if after.stats.AttacksMade > before.stats.AttacksMade {
		hit := after.level == before.level && after.enemyHealth < before.enemyHealth
		for _, event := range after.combat {
			if !event.TargetPlayer {
				hit = event.Damage != domain.Miss
				break
			}
		}
		switch {
		case hit && len(action) == 2 && action[0] == 't':
			cues = append(cues, sound.Critical) // Blade Dance instead of an ordinary hit
		case hit:
			cues = append(cues, sound.Hit)
		default:
			cues = append(cues, sound.Swing)
		}
	}
	if after.stats.HitsTaken > before.stats.HitsTaken {
		cues = append(cues, sound.Hurt)
	}
	if after.stats.EnemiesKilled > before.stats.EnemiesKilled {
		cues = append(cues, sound.Kill)
	}
	if after.stats.FoodUsed > before.stats.FoodUsed {
		cues = append(cues, sound.Heal)
	}
	if after.stats.ElixirsUsed > before.stats.ElixirsUsed {
		cues = append(cues, sound.Clarity)
	}
	if after.stats.ScrollsRead > before.stats.ScrollsRead {
		cues = append(cues, sound.Scroll)
	}
	// A parried hit clangs once per action, however many enemies were blocked.
	for _, event := range after.combat {
		if event.Parried {
			cues = append(cues, sound.Parry)
			break
		}
	}
	// Sound follows the log: a new name is a new weapon, the same name with a
	// higher bonus is the same blade sharpened (even if a stronger copy replaced it).
	if after.weaponName != before.weaponName {
		cues = append(cues, sound.Equip)
	} else if after.weaponBonus > before.weaponBonus {
		cues = append(cues, sound.Sharpen)
	}
	if len(action) == 1 && after.items > before.items {
		cues = append(cues, sound.Pickup)
	}
	return cues
}
