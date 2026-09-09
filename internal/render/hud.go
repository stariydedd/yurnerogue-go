package render

import (
	"fmt"
	"image"
	"strconv"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
	"github.com/stariydedd/yurnerogue-go/internal/domain"
)

const KeyBinds = "[WASD] Move  [F+dir] Run  [H] Weapon  [J] Food  [K] Elixir  [E] Scroll  [F1] Help  [Q] Menu"

func (r *Renderer) DrawHUD(dst *ebiten.Image, s *domain.Session) {
	l := r.Layout
	r.stonePanel(dst, image.Rect(0, l.GridTop(), l.ScreenW, l.ControlsTop()))
	if l.Touch {
		r.drawHUDTouch(dst, s)
	} else {
		r.drawHUDDesktop(dst, s)
	}
	r.drawSleepStatus(dst, s.Player)
	r.drawNotification(dst, s.Message)
}

func (r *Renderer) drawHUDDesktop(dst *ebiten.Image, s *domain.Session) {
	y, p := r.Layout.GridTop(), s.Player
	portrait := image.Rect(12, y+22, 100, y+122)
	r.uiSlot(dst, portrait, false)
	r.drawHUDPortrait(dst, portrait)
	r.Text(dst, strings.ToUpper(domain.PlayerName), r.Fonts.UI, 112, float64(y+26), uiText)
	r.drawHPBar(dst, p, image.Rect(112, y+56, 418, y+82))
	r.Text(dst, fitLabel(fmt.Sprintf("STR %d   AGI %d", p.Strength, p.Agility), r.Fonts.Compact, 308), r.Fonts.Compact, 112, float64(y+100), uiText)
	for _, x := range []int{444, 830, 1170} {
		vector.StrokeLine(dst, float32(x), float32(y+26), float32(x), float32(y+122), 1, uiEdge, false)
	}
	r.Text(dst, fitLabel(weaponLabel(p), r.Fonts.Compact, 244), r.Fonts.Compact, 568, float64(y+43), uiText)
	r.drawEffects(dst, p, image.Rect(568, y+70, 816, y+124))
	for name, rect := range HUDTargets(r.Layout) {
		if itemButtonRole[name] != "" {
			r.itemSlot(dst, rect, name, p, false)
			key := map[string]string{CtrlWeapon: "H", CtrlFood: "J", CtrlElixir: "K", CtrlScroll: "E"}[name]
			badge := image.Rect(rect.Min.X+26, y+113, rect.Max.X-26, y+136)
			r.uiSlot(dst, badge, false)
			r.slotLabel(dst, key, badge, uiText)
		} else {
			r.uiSlot(dst, rect, false)
			caption := "MENU"
			if name == CtrlSelect {
				caption = "HELP"
			}
			r.slotLabel(dst, caption, rect, uiText)
		}
	}
	status := fmt.Sprintf("LEVEL %d/%d  GOLD %d", s.LevelNum, domain.MaxLevels, p.Treasures)
	width := min(414, int(TextWidth(status, r.Fonts.Small))+28)
	box := image.Rect(r.Layout.ScreenW-12-width, y-32, r.Layout.ScreenW-12, y+1)
	r.stonePanel(dst, box)
	r.slotLabel(dst, status, box, uiText)
}

func (r *Renderer) drawHUDTouch(dst *ebiten.Image, s *domain.Session) {
	y, p := r.Layout.GridTop(), s.Player
	portrait := image.Rect(10, y+13, 66, y+85)
	r.uiSlot(dst, portrait, false)
	r.drawHUDPortrait(dst, portrait)
	r.drawHPBar(dst, p, image.Rect(76, y+16, 314, y+42))
	r.Text(dst, fitLabel(fmt.Sprintf("STR %d  AGI %d", p.Strength, p.Agility), r.Fonts.Small, 238), r.Fonts.Small, 76, float64(y+53), uiText)
	r.Text(dst, fitLabel(weaponLabel(p), r.Fonts.Small, 238), r.Fonts.Small, 76, float64(y+80), uiText)
	vector.StrokeLine(dst, 326, float32(y+16), 326, float32(y+92), 1, uiEdge, false)
	r.Text(dst, fmt.Sprintf("LEVEL %d/%d", s.LevelNum, domain.MaxLevels), r.Fonts.Small, 338, float64(y+17), uiText)
	r.Text(dst, fitLabel("GOLD "+strconv.Itoa(p.Treasures), r.Fonts.Small, 130), r.Fonts.Small, 338, float64(y+40), uiText)
	r.drawEffects(dst, p, image.Rect(338, y+58, 468, y+100))
}

func (r *Renderer) drawHPBar(dst *ebiten.Image, p *domain.Person, box image.Rectangle) {
	ratio := 0.0
	if p.MaxHealth > 0 {
		ratio = max(0, min(1, float64(p.Health)/float64(p.MaxHealth)))
	}
	fillBox(dst, box, uiInk)
	inner := box.Inset(3)
	fillBox(dst, inner, uiRecess)
	fill := inner
	fill.Max.X = fill.Min.X + int(float64(fill.Dx())*ratio)
	fillBox(dst, fill, uiAccent)
	if !fill.Empty() {
		fillBox(dst, image.Rect(fill.Min.X, fill.Min.Y, fill.Max.X, fill.Min.Y+3), uiHighlight)
	}
	strokeBox(dst, box, 1, uiEdge)
	r.slotLabel(dst, fmt.Sprintf("HP %d / %d", p.Health, p.MaxHealth), box, uiText)
}

func itemSlotValue(p *domain.Person, control string) (string, bool) {
	if p == nil {
		return "", false
	}
	if control == CtrlWeapon {
		if p.Weapon == nil {
			return "+0", true
		}
		return fmt.Sprintf("%+d", p.Weapon.StrengthEffect), true
	}
	typeOf := map[string]domain.ItemType{CtrlFood: domain.ItemFood, CtrlElixir: domain.ItemElixir, CtrlScroll: domain.ItemScroll}
	kind, ok := typeOf[control]
	if !ok {
		return "", false
	}
	count := 0
	for _, item := range p.Backpack {
		if item.Type == kind {
			count++
		}
	}
	return strconv.Itoa(count), count > 0
}

func (r *Renderer) itemSlot(dst *ebiten.Image, box image.Rectangle, control string, p *domain.Person, pressed bool) {
	r.uiSlot(dst, box, pressed)
	value, available := itemSlotValue(p, control)
	r.drawIcon(dst, itemButtonRole[control], boxCenter(box).Sub(image.Pt(0, 3)), float64(box.Dx()-18), !available)
	if value != "" {
		width := min(box.Dx()-12, int(TextWidth(value, r.Fonts.Small))+10)
		badge := image.Rect(box.Max.X-width-5, box.Max.Y-25, box.Max.X-5, box.Max.Y-5)
		fillBox(dst, badge, uiInk)
		clr := uiText
		if !available {
			clr = uiMuted
		}
		r.slotLabel(dst, value, badge, clr)
	}
}

// Stack bonuses by stat. The timer is the next expiry, when that total changes.
func statusBonuses(p *domain.Person) []domain.EffectStatus {
	var bonuses []domain.EffectStatus
	effects := p.EffectStatuses()
	for _, stat := range []domain.ItemSubType{domain.SubStrength, domain.SubAgility, domain.SubHealth} {
		bonus := domain.EffectStatus{Stat: stat}
		for _, effect := range effects {
			if effect.Stat != stat || effect.TurnsLeft <= 0 {
				continue
			}
			bonus.Amount += effect.Amount
			if bonus.TurnsLeft == 0 || effect.TurnsLeft < bonus.TurnsLeft {
				bonus.TurnsLeft = effect.TurnsLeft
			}
		}
		if bonus.TurnsLeft > 0 {
			bonuses = append(bonuses, bonus)
		}
	}
	return bonuses
}

func bonusStatLabel(stat domain.ItemSubType, compact bool) string {
	label := map[domain.ItemSubType]string{domain.SubStrength: "STR", domain.SubAgility: "AGI", domain.SubHealth: "MAX HP"}[stat]
	if compact && stat == domain.SubHealth {
		label = "MHP"
	}
	return label
}

func bonusLabel(bonus domain.EffectStatus, compact bool) string {
	return fmt.Sprintf("%s %+d %dT", bonusStatLabel(bonus.Stat, compact), bonus.Amount, bonus.TurnsLeft)
}

func (r *Renderer) drawEffects(dst *ebiten.Image, p *domain.Person, box image.Rectangle) {
	rowH := box.Dy() / 3
	for i, bonus := range statusBonuses(p) {
		label := fitLabel(bonusLabel(bonus, r.Layout.Touch), r.Fonts.Small, float64(box.Dx()))
		x, y := float64(box.Min.X), float64(box.Min.Y+i*rowH)
		r.Text(dst, label, r.Fonts.Small, x, y, uiText)
		r.Text(dst, bonusStatLabel(bonus.Stat, r.Layout.Touch), r.Fonts.Small, x, y, uiAccent)
	}
}

func (r *Renderer) drawSleepStatus(dst *ebiten.Image, p *domain.Person) {
	if !p.Sleeping || p.SleepTurns <= 0 {
		return
	}
	x, y := 286, r.Layout.GridTop()+28
	label := fmt.Sprintf("SLEEP %dT", p.SleepTurns)
	if r.Layout.Touch {
		x, y, label = 12, r.Layout.GridTop()+90, fmt.Sprintf("ZZ %dT", p.SleepTurns)
	}
	r.Text(dst, label, r.Fonts.Small, float64(x), float64(y), uiAccent)
}

func (r *Renderer) drawNotification(dst *ebiten.Image, message string) {
	if strings.TrimSpace(message) == "" {
		return
	}
	maxWidth := 794
	if r.Layout.Touch {
		maxWidth = r.Layout.ScreenW - 24
	}
	lines := messageLines(message, (maxWidth-28)/10)
	if len(lines) > 2 {
		lines = []string{lines[0], strings.Join(lines[1:], " ")}
	}
	width := 0
	for i, line := range lines {
		lines[i] = fitLabel(line, r.Fonts.Small, float64(maxWidth-28))
		width = max(width, int(TextWidth(lines[i], r.Fonts.Small)))
	}
	height := 24 + len(lines)*18
	box := image.Rect(12, r.Layout.GridH-height-8, 12+width+28, r.Layout.GridH-8)
	r.stonePanel(dst, box)
	for i, line := range lines {
		r.Text(dst, line, r.Fonts.Small, float64(box.Min.X+14), float64(box.Min.Y+12+i*18), uiText)
	}
}

func weaponLabel(p *domain.Person) string {
	if p.Weapon == nil {
		return domain.BaseWeaponName
	}
	return p.Weapon.Name + " +" + strconv.Itoa(p.Weapon.StrengthEffect)
}

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
