package render

import (
	"fmt"
	"image"
	"image/color"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
	"github.com/stariydedd/yurnerogue-go/internal/domain"
)

// hudCache keeps the HUD drawn for one game state. Its panels are vector
// strokes and shaped text, too slow to rebuild every frame on a weak machine:
// the long frames starved the browser audio buffer. Only the portrait is
// animated, so it is drawn on top every frame.
type hudCache struct {
	key      uint64
	drawn    bool
	frame    *ebiten.Image
	portrait image.Rectangle
	caching  bool
}

// hudHash is an FNV-1a hash that takes the HUD's fields without formatting
// them into a string every frame.
type hudHash uint64

func (h *hudHash) int(v int) {
	for i := 0; i < 8; i++ {
		*h = (*h ^ hudHash(byte(v>>(8*i)))) * 1099511628211
	}
}

func (h *hudHash) bool(v bool) {
	if v {
		h.int(1)
	} else {
		h.int(0)
	}
}

func (h *hudHash) str(s string) {
	h.int(len(s))
	for i := 0; i < len(s); i++ {
		*h = (*h ^ hudHash(s[i])) * 1099511628211
	}
}

// hudKey hashes everything the HUD shows. A state that changes the HUD must
// change this key, or the HUD would stay stale.
func (r *Renderer) hudKey(s *domain.Session) uint64 {
	p, l := s.Player, r.Layout
	h := hudHash(14695981039346656037)
	h.str(string(l.Language))
	for _, v := range []int{l.ScreenW, l.ScreenH, l.GridW, l.GridH, l.PanelH, l.ControlsH,
		s.LevelNum, p.Treasures, p.Health, p.MaxHealth, p.Strength, p.Agility, p.SleepTurns,
		p.StrikeCooldown, p.GuardCooldown} {
		h.int(v)
	}
	for _, v := range []bool{l.Touch, p.Sleeping, p.StrikeArmed, p.Guarding} {
		h.bool(v)
	}
	for _, e := range p.EffectStatuses() {
		h.int(int(e.Stat))
		h.int(e.Amount)
		h.int(e.TurnsLeft)
	}
	h.bool(p.Weapon != nil)
	if p.Weapon != nil {
		h.str(p.Weapon.Name)
		h.int(p.Weapon.StrengthEffect)
	}
	h.int(len(p.Backpack))
	for _, it := range p.Backpack {
		h.int(int(it.Type))
		h.str(it.Name)
	}
	h.str(s.Message)
	h.int(len(s.EventLog))
	for _, line := range s.EventLog {
		h.str(line)
	}
	return uint64(h)
}

func (r *Renderer) DrawHUD(dst *ebiten.Image, s *domain.Session) {
	key, size := r.hudKey(s), dst.Bounds().Size()
	if r.hud == nil || r.hud.frame.Bounds().Size() != size {
		if r.hud != nil {
			r.hud.frame.Deallocate()
		}
		r.hud = &hudCache{frame: ebiten.NewImage(size.X, size.Y)}
	}
	if r.hud.key != key || !r.hud.drawn {
		r.hud.frame.Clear()
		r.hud.caching = true
		r.drawHUDLayers(r.hud.frame, s)
		r.hud.caching = false
		r.hud.key, r.hud.drawn = key, true
	}
	dst.DrawImage(r.hud.frame, nil)
	r.drawHUDPortrait(dst, r.hud.portrait)
}

func (r *Renderer) drawHUDLayers(dst *ebiten.Image, s *domain.Session) {
	l := r.Layout
	r.stonePanel(dst, image.Rect(0, l.GridTop(), l.ScreenW, l.ControlsTop()))
	if l.Touch {
		r.drawHUDTouch(dst, s)
	} else {
		r.drawHUDDesktop(dst, s)
	}
}

func (r *Renderer) drawHUDDesktop(dst *ebiten.Image, s *domain.Session) {
	y, p := r.Layout.GridTop(), s.Player
	h := desktopHUDGeometry(r.Layout)
	r.uiSlot(dst, h.portrait, false)
	r.drawHUDPortrait(dst, h.portrait)
	r.drawInventoryHUD(dst, p)
	r.drawHPBar(dst, p, h.health)
	r.Text(dst, r.hudStatsLabel(p, r.Fonts.Compact, float64(h.health.Dx()), "STRENGTH %d   AGILITY %d"), r.Fonts.Compact, float64(h.health.Min.X), float64(y+56), uiText)
	for _, divider := range []int{h.health.Max.X + 16, h.status.Min.X - 12} {
		vector.StrokeLine(dst, float32(divider), float32(h.health.Min.Y), float32(divider), float32(h.portrait.Max.Y), 1, uiEdge, false)
	}
	for name, rect := range h.targets {
		if role := abilityButtonRole[name]; role != "" {
			r.abilitySlot(dst, rect, name, p, false)
			key := "F"
			if name == CtrlGuard {
				key = "E"
			}
			badge := h.keyBadge(rect)
			r.uiSlot(dst, badge, false)
			r.slotLabel(dst, key, badge, uiText)
			continue
		}
		if itemButtonRole[name] != "" {
			r.itemSlot(dst, rect, name, p, false)
			key := map[string]string{CtrlFood: "C", CtrlElixir: "X"}[name]
			badge := h.keyBadge(rect)
			r.uiSlot(dst, badge, false)
			r.slotLabel(dst, key, badge, uiText)
		} else {
			r.uiSlot(dst, rect, false)
			caption := "MENU"
			if name == CtrlSelect {
				caption = "HELP"
				if r.tr("HELP") != "HELP" {
					caption = "HUD HELP"
				}
			}
			r.slotLabel(dst, r.tr(caption), rect, uiText)
		}
	}
	level := fmt.Sprintf(r.tr("LEVEL %d/%d"), s.LevelNum, domain.MaxLevels)
	gold := r.tr("GOLD") + " " + strconv.Itoa(p.Treasures)
	label := fitLabel(level+"   "+gold, r.Fonts.Small, float64(h.status.Dx()))
	r.Text(dst, label, r.Fonts.Small, float64(h.status.Min.X), float64(h.status.Min.Y), uiText)
	x, effectY := float64(h.status.Min.X)+TextWidth(label, r.Fonts.Small)+20, float64(h.status.Min.Y)
	for _, effect := range r.hudEffects(p) {
		head, turns := effect.head, effect.turns
		width := TextWidth(head+" "+turns, r.Fonts.Small)
		if x+width > float64(h.effects.Max.X) && effectY < float64(h.effects.Min.Y) {
			x, effectY = float64(h.effects.Min.X), float64(h.effects.Min.Y)
		}
		x += r.drawEffect(dst, head, turns, x, effectY, max(0, float64(h.effects.Max.X)-x), effect.tint) + 10
	}
	r.drawEventLog(dst, s, h.log)
}

func (r *Renderer) drawHUDTouch(dst *ebiten.Image, s *domain.Session) {
	y, p := r.Layout.GridTop(), s.Player
	portrait := image.Rect(10, y+13, 66, y+85)
	r.uiSlot(dst, portrait, false)
	r.drawHUDPortrait(dst, portrait)
	r.drawHPBar(dst, p, image.Rect(76, y+16, 314, y+42))
	r.Text(dst, r.hudStatsLabel(p, r.Fonts.Small, 238, "STRENGTH %d  AGILITY %d"), r.Fonts.Small, 76, float64(y+53), uiText)
	r.drawInventoryHUD(dst, p)
	vector.StrokeLine(dst, 326, float32(y+16), 326, float32(y+92), 1, uiEdge, false)
	r.Text(dst, fmt.Sprintf(r.tr("LEVEL %d/%d"), s.LevelNum, domain.MaxLevels), r.Fonts.Small, 338, float64(y+17), uiText)
	r.Text(dst, fitLabel(r.tr("GOLD")+" "+strconv.Itoa(p.Treasures), r.Fonts.Small, 130), r.Fonts.Small, 338, float64(y+40), uiText)
	r.drawEffects(dst, p, image.Rect(338, y+54, 468, y+102))
	r.drawEventLog(dst, s, image.Rect(10, y+104, r.Layout.ScreenW-10, r.Layout.ControlsTop()-8))
}

func (r *Renderer) abilitySlot(dst *ebiten.Image, box image.Rectangle, control string, p *domain.Person, pressed bool) {
	cooldown, armed := 0, false
	if p != nil {
		cooldown = p.GuardCooldown
		if control == CtrlStrike {
			cooldown, armed = p.StrikeCooldown, p.StrikeArmed
		}
	}
	r.uiSlot(dst, box, pressed || armed)
	r.drawIcon(dst, abilityButtonRole[control], boxCenter(box).Sub(image.Pt(0, 3)), float64(min(box.Dx(), box.Dy())-14), p == nil || cooldown > 0 || p.Sleeping)
	if cooldown > 0 {
		badge := image.Rect(box.Max.X-28, box.Max.Y-24, box.Max.X-4, box.Max.Y-4)
		fillBox(dst, badge, uiInk)
		r.slotLabel(dst, strconv.Itoa(cooldown), badge, uiText)
	}
}

func (r *Renderer) drawInventoryHUD(dst *ebiten.Image, p *domain.Person) {
	y := r.Layout.GridTop()
	weapon := image.Rect(116, y+74, 418, y+110)
	face := r.Fonts.Compact
	if r.Layout.Touch {
		face = r.Fonts.Small
		weapon = image.Rect(76, y+66, 314, y+108)
	}
	r.Text(dst, fitLabel(weaponLabel(p), face, float64(weapon.Dx())), face, float64(weapon.Min.X), float64(weapon.Min.Y+14), uiText)
}

func (r *Renderer) hudStatsLabel(p *domain.Person, face text.Face, width float64, format string) string {
	label := fmt.Sprintf(r.tr(format), p.Strength, p.Agility)
	if TextWidth(label, face) > width {
		label = fmt.Sprintf("%s %d  %s %d", r.tr("STR"), p.Strength, r.tr("AGI"), p.Agility)
	}
	return fitLabel(label, face, width)
}

func (r *Renderer) hudHealthLabel(p *domain.Person, width float64) string {
	label := fmt.Sprintf(r.tr("HEALTH %d / %d"), p.Health, p.MaxHealth)
	if TextWidth(label, r.Fonts.Small) > width {
		label = fmt.Sprintf("%s %d / %d", r.tr("HP"), p.Health, p.MaxHealth)
	}
	return label
}

func healthBarColors(p *domain.Person) (fill, highlight, edge color.RGBA) {
	if p != nil && p.MaxHealth > 0 && float64(p.Health)/float64(p.MaxHealth) < 0.25 {
		return uiDanger, color.RGBA{255, 117, 104, 255}, uiDanger
	}
	return uiAccent, uiHighlight, uiEdge
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
	fillColor, highlight, edge := healthBarColors(p)
	fillBox(dst, fill, fillColor)
	if !fill.Empty() {
		fillBox(dst, image.Rect(fill.Min.X, fill.Min.Y, fill.Max.X, fill.Min.Y+3), highlight)
	}
	strokeBox(dst, box, 1, edge)
	r.slotLabel(dst, r.hudHealthLabel(p, float64(box.Dx()-8)), box, uiText)
}

func itemSlotValue(p *domain.Person, control string) (string, bool) {
	if p == nil {
		return "", false
	}
	typeOf := map[string]domain.ItemType{CtrlFood: domain.ItemFood, CtrlElixir: domain.ItemElixir}
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
	effects := r.hudEffects(p)
	rowH := box.Dy() / max(3, len(effects))
	for i, effect := range effects {
		r.drawEffect(dst, effect.head, effect.turns, float64(box.Min.X), float64(box.Min.Y+i*rowH), float64(box.Dx()), effect.tint)
	}
}

type hudEffect struct {
	head, turns string
	tint        color.RGBA
}

func (r *Renderer) hudEffects(p *domain.Person) []hudEffect {
	var effects []hudEffect
	// Sleep comes first so it remains visible even when the status row is crowded.
	if p.Sleeping && p.SleepTurns > 0 {
		effects = append(effects, hudEffect{r.tr("SLEEP"), fmt.Sprintf(r.tr("%dT"), p.SleepTurns), uiDebuff})
	}
	for _, bonus := range statusBonuses(p) {
		head, turns := r.bonusParts(bonus)
		effects = append(effects, hudEffect{head, turns, uiAccent})
	}
	return effects
}

// bonusParts splits a bonus into the stat and amount ("STR +3") and the
// remaining turns ("20T"), translated for the current language.
func (r *Renderer) bonusParts(bonus domain.EffectStatus) (string, string) {
	return fmt.Sprintf("%s %+d", r.tr(bonusStatLabel(bonus.Stat, true)), bonus.Amount),
		fmt.Sprintf(r.tr("%dT"), bonus.TurnsLeft)
}

// drawBonus is the single bonus style of both HUDs: stat and amount in the
// accent colour, turns in the text colour. Returns the drawn width.
func (r *Renderer) drawBonus(dst *ebiten.Image, bonus domain.EffectStatus, x, y, width float64) float64 {
	head, turns := r.bonusParts(bonus)
	return r.drawEffect(dst, head, turns, x, y, width, uiAccent)
}

func (r *Renderer) drawEffect(dst *ebiten.Image, head, turns string, x, y, width float64, tint color.RGBA) float64 {
	face := r.Fonts.Small
	if full := head + " " + turns; TextWidth(full, face) > width {
		label := fitLabel(full, face, width)
		r.Text(dst, label, face, x, y, tint)
		return TextWidth(label, face)
	}
	r.Text(dst, head, face, x, y, tint)
	r.Text(dst, turns, face, x+TextWidth(head+" ", face), y, uiText)
	return TextWidth(head+" "+turns, face)
}

func weaponLabel(p *domain.Person) string {
	if p.Weapon == nil {
		return domain.BaseWeaponName
	}
	return p.Weapon.Name // the name already shows how strong it is
}

func wrapText(s string, maxChars int) []string {
	var lines []string
	current := ""
	for _, word := range strings.Fields(s) {
		candidate := word
		if current != "" {
			candidate = current + " " + word
		}
		if utf8.RuneCountInString(candidate) > maxChars && current != "" {
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
