package game

import (
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/stariydedd/yurnerogue-go/internal/domain"
	"github.com/stariydedd/yurnerogue-go/internal/sound"
)

// Звук включает только интерактивная точка входа; проверка повторов остаётся беззвучной.
func (g *Game) EnableAudio() {
	if g.audio == nil {
		g.audio = sound.New()
	}
}

func (g *Game) updateAudio(before State) {
	g.audio.SilenceMusic(g.state == StateWin || g.state == StateDeath)
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
	// Та же геометрия, что у PathCells в рендерере, независимо от предметов и врагов
	// поверх пола и от того, какие соседние клетки уже открыты.
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
	// Бег проходит несколько клеток мгновенно: один звук приземления, а не очередь
	// шагов после того, как игрок уже встал. Телепорты не шаги.
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
			cues = append(cues, sound.Critical) // Blade Dance вместо обычного удара
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
	// Парированный удар звенит один раз за действие, сколько бы врагов ни было заблокировано.
	for _, event := range after.combat {
		if event.Parried {
			cues = append(cues, sound.Parry)
			break
		}
	}
	// Звук следует за журналом: новое название означает новое оружие, то же название
	// с большим бонусом означает заточку того же клинка (даже если его заменила более сильная копия).
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
