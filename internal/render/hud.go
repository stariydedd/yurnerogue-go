package render

import (
	"strconv"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"

	"github.com/stariydedd/yurnerogue-go/internal/domain"
)

// KeyBinds — подсказка по управлению в нижней строке панели (только десктоп).
const KeyBinds = "[WASD] Move  [F+dir] Run  [H] Weapon  [J] Food  [K] Elixir  [E] Scroll  [F1] Help  [Q] Menu"

// DrawHUD рисует статус-панель под игровым полем.
func (r *Renderer) DrawHUD(screen *ebiten.Image, s *domain.Session) {
	l := r.Layout
	top := float64(l.GridTop())
	vector.DrawFilledRect(screen, 0, float32(top), float32(l.ScreenW), float32(l.PanelH), Black, false)

	if l.Touch {
		r.drawHUDTouch(screen, s, top)
		return
	}
	r.drawHUDDesktop(screen, s, top)
}

// drawHUDDesktop — широкая раскладка: портрет в слоте постоянной ширины,
// три ряда текста, сообщение и подсказка по клавишам справа.
func (r *Renderer) drawHUDDesktop(screen *ebiten.Image, s *domain.Session, top float64) {
	l := r.Layout
	p := s.Player

	r.drawPortrait(screen, top+float64(l.PanelH)/2)
	textX := 12.0 + portraitSlot + 14

	r.Text(screen, domain.PlayerName, r.Fonts.UI, textX, top+8, Gold)
	r.drawHPBar(screen, p, textX, top+34, 200)

	stats := "STR " + strconv.Itoa(p.Strength) +
		"   AGI " + strconv.Itoa(p.Agility) +
		"   LVL " + strconv.Itoa(s.LevelNum) +
		"   GOLD " + strconv.Itoa(p.Treasures) +
		"   WPN " + weaponLabel(p)
	r.Text(screen, stats, r.Fonts.UI, textX, top+62, White)

	if s.Message != "" {
		r.TextRight(screen, s.Message, r.Fonts.UI, float64(l.ScreenW)-12, top+10, MsgColor)
	}
	r.TextRight(screen, KeyBinds, r.Fonts.Small, float64(l.ScreenW)-12, float64(l.ScreenH)-14, HintColor)
}

// drawHUDTouch — узкая раскладка: оружию отдельная строка, сообщению две,
// подсказки по клавишам не нужны — управление экранное.
func (r *Renderer) drawHUDTouch(screen *ebiten.Image, s *domain.Session, top float64) {
	l := r.Layout
	p := s.Player

	r.drawPortrait(screen, top+float64(l.PanelH)/2)
	textX := 12.0 + portraitSlot + 14

	r.Text(screen, domain.PlayerName, r.Fonts.UI, textX, top+8, Gold)
	r.TextRight(screen, "LVL "+strconv.Itoa(s.LevelNum), r.Fonts.UI, float64(l.ScreenW)-12, top+8, Gold)

	r.drawHPBar(screen, p, textX, top+34, 180)

	r.Text(screen, "STR "+strconv.Itoa(p.Strength)+"  AGI "+strconv.Itoa(p.Agility),
		r.Fonts.UI, textX, top+62, White)
	r.TextRight(screen, "GOLD "+strconv.Itoa(p.Treasures), r.Fonts.UI, float64(l.ScreenW)-12, top+62, White)

	r.Text(screen, "WPN "+weaponLabel(p), r.Fonts.Small, textX, top+94, HintColor)

	if s.Message != "" {
		// Сообщения начинаются от левого края: они ниже портрета.
		maxChars := (l.ScreenW - 20) / 10
		for i, line := range messageLines(s.Message, maxChars) {
			if i >= 2 {
				break
			}
			r.Text(screen, line, r.Fonts.Small, 12, top+118+float64(i)*22, MsgColor)
		}
	}
}

// portraitSlot — ширина слота портрета: постоянная, чтобы кадры анимации
// не сдвигали остальную вёрстку.
const portraitSlot = 56.0

func (r *Renderer) drawPortrait(screen *ebiten.Image, centerY float64) {
	img := r.sprites.Frame("player", r.AnimTick())
	if img == nil {
		return
	}
	w, h := img.Bounds().Dx(), img.Bounds().Dy()
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(12+portraitSlot/2-float64(w)/2, centerY-float64(h)/2)
	screen.DrawImage(img, op)
}

// drawHPBar рисует сердечко и полосу здоровья.
func (r *Renderer) drawHPBar(screen *ebiten.Image, p *domain.Person, x, y, barW float64) {
	const barH = 16.0

	if heart := r.sprites.Frame("heart", 0); heart != nil {
		hw, hh := heart.Bounds().Dx(), heart.Bounds().Dy()
		op := &ebiten.DrawImageOptions{}
		op.GeoM.Scale(18/float64(hw), 18/float64(hh))
		op.GeoM.Translate(x, y-1)
		screen.DrawImage(heart, op)
	}

	barX := x + 26
	ratio := 0.0
	if p.MaxHealth > 0 {
		ratio = float64(p.Health) / float64(p.MaxHealth)
	}
	ratio = max(0, min(1, ratio))

	vector.DrawFilledRect(screen, float32(barX), float32(y), float32(barW), barH, HPBack, false)
	vector.DrawFilledRect(screen, float32(barX), float32(y), float32(barW*ratio), barH, HPFill, false)
	vector.StrokeRect(screen, float32(barX), float32(y), float32(barW), barH, 2, HPBorder, false)

	hp := strconv.Itoa(p.Health) + "/" + strconv.Itoa(p.MaxHealth)
	r.Text(screen, hp, r.Fonts.Small, barX+barW+10, y+(barH-10)/2, White)
}

// weaponLabel — название оружия с бонусом или «Bare hands».
func weaponLabel(p *domain.Person) string {
	if p.Weapon == nil {
		return "Bare hands"
	}
	return p.Weapon.Name + " +" + strconv.Itoa(p.Weapon.StrengthEffect)
}

// messageLines делит сообщение на строки для узкого экрана: «Picked up: имя»
// ломается по двоеточию, остальное переносится по словам.
func messageLines(msg string, maxChars int) []string {
	if len(msg) <= maxChars {
		return []string{msg}
	}
	if head, tail, ok := strings.Cut(msg, ": "); ok {
		if len(head)+1 <= maxChars && len(tail) <= maxChars {
			return []string{head + ":", tail}
		}
	}
	return wrapText(msg, maxChars)
}

// wrapText переносит строку по словам.
func wrapText(s string, maxChars int) []string {
	var lines []string
	current := ""
	for _, word := range strings.Fields(s) {
		candidate := word
		if current != "" {
			candidate = current + " " + word
		}
		if len(candidate) > maxChars && current != "" {
			lines = append(lines, current)
			current = word
			continue
		}
		current = candidate
	}
	if current != "" {
		lines = append(lines, current)
	}
	return lines
}
