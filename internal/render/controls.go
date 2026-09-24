package render

import (
	"image"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
	"github.com/stariydedd/yurnerogue-go/internal/domain"
)

const (
	CtrlUp     = "up"
	CtrlDown   = "down"
	CtrlLeft   = "left"
	CtrlRight  = "right"
	CtrlRun    = "run"
	CtrlFood   = "food"
	CtrlElixir = "elixir"
	CtrlStrike = "strike"
	CtrlGuard  = "guard"
	CtrlWait   = "wait"
	CtrlMenu   = "menu"
	CtrlSelect = "select"
)

var DPadControls = map[string]bool{CtrlUp: true, CtrlDown: true, CtrlLeft: true, CtrlRight: true}
var itemButtonRole = map[string]string{CtrlFood: "food", CtrlElixir: "elixir"}
var abilityButtonRole = map[string]string{CtrlStrike: "special-strike", CtrlGuard: "defense"}

// Controls shares the exact, non-overlapping geometry between drawing and input.
type Controls struct {
	Panel   image.Rectangle
	targets map[string]image.Rectangle
	hub     image.Rectangle
}

func NewControls(l Layout) *Controls {
	top := l.ControlsTop()
	c := &Controls{
		Panel:   image.Rect(0, top, l.ScreenW, top+l.ControlsH),
		targets: map[string]image.Rectangle{},
		hub:     image.Rect(74, top+84, 134, top+144),
	}
	c.targets[CtrlUp] = image.Rect(74, top+24, 134, top+84)
	c.targets[CtrlDown] = image.Rect(74, top+144, 134, top+204)
	c.targets[CtrlLeft] = image.Rect(14, top+84, 74, top+144)
	c.targets[CtrlRight] = image.Rect(134, top+84, 194, top+144)
	c.targets[CtrlWait] = c.hub.Inset(8) // leave a dead zone around the wait button
	c.targets[CtrlRun] = image.Rect(216, top+22, 280, top+82)
	c.targets[CtrlMenu] = image.Rect(212, top+100, 284, top+146)
	c.targets[CtrlSelect] = image.Rect(212, top+160, 284, top+206)
	for i, name := range []string{CtrlStrike, CtrlFood, CtrlGuard, CtrlElixir} {
		x, y := 297+(i%2)*88, top+23+(i/2)*92
		c.targets[name] = image.Rect(x, y, x+78, y+78)
	}
	return c
}

func controlAt(targets map[string]image.Rectangle, x, y int) string {
	for name, rect := range targets {
		if image.Pt(x, y).In(rect) {
			return name
		}
	}
	return ""
}

func (c *Controls) ControlAt(x, y int) string {
	return controlAt(c.targets, x, y)
}

// HUDTargets are clickable desktop slots; touch controls live below the HUD.
func HUDTargets(l Layout) map[string]image.Rectangle {
	if l.Touch {
		return nil
	}
	return desktopHUDGeometry(l).targets
}

type desktopHUDLayout struct {
	portrait, health, inventory, effects, status, log image.Rectangle
	targets                                           map[string]image.Rectangle
}

func desktopHUDGeometry(l Layout) desktopHUDLayout {
	y := l.GridTop()
	heroRight := 418
	statusLeft := 800
	h := desktopHUDLayout{
		portrait:  image.Rect(16, y+14, 104, y+114),
		health:    image.Rect(116, y+16, heroRight, y+42),
		inventory: image.Rect(heroRight+32, y+12, 768, y+124),
		effects:   image.Rect(statusLeft, y+28, l.ScreenW-112, y+38),
		status:    image.Rect(statusLeft, y+16, l.ScreenW-112, y+28),
		log:       image.Rect(statusLeft, y+38, l.ScreenW-112, y+114),
		targets: map[string]image.Rectangle{
			CtrlMenu:   image.Rect(l.ScreenW-88, y+16, l.ScreenW-16, y+56),
			CtrlSelect: image.Rect(l.ScreenW-88, y+74, l.ScreenW-16, y+114),
		},
	}
	const slotSize = 72
	stride := 82
	for i, name := range []string{CtrlStrike, CtrlGuard, CtrlFood, CtrlElixir} {
		x := h.inventory.Min.X + i*stride
		h.targets[name] = image.Rect(x, h.portrait.Min.Y, x+slotSize, h.portrait.Min.Y+slotSize)
	}
	return h
}

func (h desktopHUDLayout) keyBadge(slot image.Rectangle) image.Rectangle {
	bottom := h.portrait.Max.Y
	center := slot.Min.X + slot.Dx()/2
	return image.Rect(center-14, bottom-20, center+14, bottom)
}

func HUDControlAt(l Layout, x, y int) string { return controlAt(HUDTargets(l), x, y) }

type SelectLabel int

const (
	SelectConfirm SelectLabel = iota
	SelectHelp
	SelectUse
)

func (c *Controls) Draw(dst *ebiten.Image, r *Renderer, pressed map[string]bool, label SelectLabel, showRun bool, player *domain.Person) {
	r.stonePanel(dst, c.Panel)
	for name, rect := range c.targets {
		if role := abilityButtonRole[name]; role != "" {
			r.abilitySlot(dst, rect, name, player, pressed[name])
			continue
		}
		if name == CtrlRun && !showRun {
			continue
		}
		if role := itemButtonRole[name]; role != "" {
			r.itemSlot(dst, rect, name, player, pressed[name])
			continue
		}
		box := rect
		if DPadControls[name] {
			box = box.Inset(3)
		}
		r.uiSlot(dst, box, pressed[name])
		if DPadControls[name] {
			drawArrow(dst, name, box)
			continue
		}
		switch name {
		case CtrlRun:
			r.drawCentered(dst, "ui_run", boxCenter(rect).Sub(image.Pt(0, 8)), 28)
			r.slotLabel(dst, r.tr("RUN"), image.Rect(rect.Min.X, rect.Max.Y-23, rect.Max.X, rect.Max.Y-3), uiText)
		case CtrlWait:
			r.slotLabel(dst, "Zzz", rect, uiText)
		case CtrlMenu:
			r.slotLabel(dst, r.tr("MENU"), rect, uiText)
		case CtrlSelect:
			caption := "SELECT"
			if label == SelectHelp {
				caption = "HELP"
				if r.tr("HELP") != "HELP" {
					caption = "HUD HELP"
				}
			}
			if label == SelectUse {
				caption = "USE"
			}
			r.slotLabel(dst, r.tr(caption), rect, uiText)
		}
	}
}

func (r *Renderer) drawCentered(dst *ebiten.Image, role string, center image.Point, size float64) {
	r.drawIcon(dst, role, center, size, false)
}

func (r *Renderer) drawIcon(dst *ebiten.Image, role string, center image.Point, size float64, dim bool) {
	img := r.sprites.Frame(role, 0)
	if img == nil {
		return
	}
	w, h := float64(img.Bounds().Dx()), float64(img.Bounds().Dy())
	scale := min(size/w, size/h)
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Scale(scale, scale)
	op.GeoM.Translate(float64(center.X)-w*scale/2, float64(center.Y)-h*scale/2)
	if dim {
		op.ColorScale.Scale(0.38, 0.42, 0.45, 1)
	}
	dst.DrawImage(img, op)
}

func drawArrow(dst *ebiten.Image, dir string, rect image.Rectangle) {
	p := boxCenter(rect)
	const steps, step = 6, 2
	for i := 0; i < steps; i++ {
		half := float32((i + 1) * step)
		var x, y, w, h float32
		switch dir {
		case CtrlUp:
			x, y, w, h = float32(p.X)-half, float32(p.Y-steps+i*2), half*2, step
		case CtrlDown:
			x, y, w, h = float32(p.X)-half, float32(p.Y+steps-i*2-step), half*2, step
		case CtrlLeft:
			x, y, w, h = float32(p.X-steps+i*2), float32(p.Y)-half, step, half*2
		case CtrlRight:
			x, y, w, h = float32(p.X+steps-i*2-step), float32(p.Y)-half, step, half*2
		}
		vector.DrawFilledRect(dst, x, y, w, h, uiText, false)
	}
}
