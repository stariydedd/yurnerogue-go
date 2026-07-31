package render

import (
	"image"
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

// Названия экранных контролов. Ими же game адресует нажатия.
const (
	CtrlUp     = "up"
	CtrlDown   = "down"
	CtrlLeft   = "left"
	CtrlRight  = "right"
	CtrlRun    = "run"
	CtrlWeapon = "weapon"
	CtrlFood   = "food"
	CtrlElixir = "elixir"
	CtrlScroll = "scroll"
	CtrlMenu   = "menu"
	CtrlSelect = "select"
)

// DPadControls — направления крестовины.
var DPadControls = map[string]bool{
	CtrlUp: true, CtrlDown: true, CtrlLeft: true, CtrlRight: true,
}

// itemButtonRole — спрайт на кнопке предмета.
var itemButtonRole = map[string]string{
	CtrlWeapon: "sword",
	CtrlFood:   "food",
	CtrlElixir: "elixir",
	CtrlScroll: "scroll",
}

// Controls — геометрия панели экранных кнопок в стиле ретро-консоли:
// слева крестовина с бегом в центре, справа ромб предметов, внизу MENU
// и SELECT.
type Controls struct {
	Panel image.Rectangle

	dpad     map[string]image.Rectangle
	hub      image.Rectangle
	dpadCent image.Point

	buttons   map[string]image.Point
	btnRadius int

	pills map[string]image.Rectangle
}

// NewControls раскладывает кнопки по нижней панели.
func NewControls(l Layout) *Controls {
	top := l.ControlsTop()
	c := &Controls{
		Panel:     image.Rect(0, top, l.ScreenW, top+l.ControlsH),
		dpad:      map[string]image.Rectangle{},
		buttons:   map[string]image.Point{},
		pills:     map[string]image.Rectangle{},
		btnRadius: 34,
	}

	const cell = 52
	cx, cy := 110, top+int(float64(l.ControlsH)*0.40)
	c.dpadCent = image.Pt(cx, cy)
	c.hub = image.Rect(cx-cell/2, cy-cell/2, cx+cell/2, cy+cell/2)
	c.dpad[CtrlUp] = image.Rect(cx-cell/2, cy-cell/2-cell, cx+cell/2, cy-cell/2)
	c.dpad[CtrlDown] = image.Rect(cx-cell/2, cy+cell/2, cx+cell/2, cy+cell/2+cell)
	c.dpad[CtrlLeft] = image.Rect(cx-cell/2-cell, cy-cell/2, cx-cell/2, cy+cell/2)
	c.dpad[CtrlRight] = image.Rect(cx+cell/2, cy-cell/2, cx+cell/2+cell, cy+cell/2)

	bx, by := l.ScreenW-110, cy
	const off = 56
	c.buttons[CtrlWeapon] = image.Pt(bx, by-off)
	c.buttons[CtrlFood] = image.Pt(bx-off, by)
	c.buttons[CtrlElixir] = image.Pt(bx+off, by)
	c.buttons[CtrlScroll] = image.Pt(bx, by+off)

	const pillW, pillH = 96, 36
	py := top + l.ControlsH - 52
	c.pills[CtrlMenu] = image.Rect(l.ScreenW/2-8-pillW, py-pillH/2, l.ScreenW/2-8, py+pillH/2)
	c.pills[CtrlSelect] = image.Rect(l.ScreenW/2+8, py-pillH/2, l.ScreenW/2+8+pillW, py+pillH/2)
	return c
}

// ControlAt возвращает контрол под точкой или пустую строку.
func (c *Controls) ControlAt(x, y int) string {
	p := image.Pt(x, y)
	for name, rect := range c.dpad {
		if p.In(rect.Inset(-6)) {
			return name
		}
	}
	if p.In(c.hub) {
		return CtrlRun
	}
	for name, center := range c.buttons {
		dx, dy := float64(x-center.X), float64(y-center.Y)
		if math.Hypot(dx, dy) <= float64(c.btnRadius+8) {
			return name
		}
	}
	for name, rect := range c.pills {
		if p.In(rect.Inset(-6)) {
			return name
		}
	}
	return ""
}

// SelectLabel — подпись контекстной кнопки: в игре справка, в списке
// предметов использование, иначе подтверждение.
type SelectLabel int

const (
	SelectConfirm SelectLabel = iota
	SelectHelp
	SelectUse
)

// Draw рисует панель кнопок. pressed отмечает нажатые контролы.
func (c *Controls) Draw(screen *ebiten.Image, r *Renderer, pressed map[string]bool, label SelectLabel) {
	p := c.Panel
	vector.DrawFilledRect(screen, float32(p.Min.X), float32(p.Min.Y),
		float32(p.Dx()), float32(p.Dy()), PanelBG, false)
	vector.StrokeLine(screen, float32(p.Min.X), float32(p.Min.Y),
		float32(p.Max.X), float32(p.Min.Y), 2, BtnEdge, false)

	// Крестовина: цельный крест, поверх стрелки, в центре кнопка бега.
	for name, rect := range c.dpad {
		fill := BtnBase
		if pressed[name] {
			fill = BtnPressed
		}
		drawRoundRect(screen, rect, fill)
	}
	drawRoundRect(screen, c.hub, BtnBase)
	for name, rect := range c.dpad {
		strokeRoundRect(screen, rect, BtnEdge)
		c.drawArrow(screen, name, rect)
	}

	hubFill := PanelBG
	if pressed[CtrlRun] {
		hubFill = BtnPressed
	}
	vector.DrawFilledCircle(screen, float32(c.dpadCent.X), float32(c.dpadCent.Y), 22, hubFill, true)
	vector.StrokeCircle(screen, float32(c.dpadCent.X), float32(c.dpadCent.Y), 22, 2, BtnEdge, true)
	r.drawCentered(screen, "ui_run", c.dpadCent, 34)

	// Ромб предметов со спрайтами.
	for name, center := range c.buttons {
		fill := BtnBase
		if pressed[name] {
			fill = BtnPressed
		}
		vector.DrawFilledCircle(screen, float32(center.X), float32(center.Y), float32(c.btnRadius), fill, true)
		vector.StrokeCircle(screen, float32(center.X), float32(center.Y), float32(c.btnRadius), 3, BtnEdge, true)
		r.drawCentered(screen, itemButtonRole[name], center, float64(c.btnRadius*2-22))
	}

	for name, rect := range c.pills {
		fill := BtnBase
		if pressed[name] {
			fill = BtnPressed
		}
		drawRoundRect(screen, rect, fill)
		strokeRoundRect(screen, rect, BtnEdge)

		text, clr := "MENU", White
		if name == CtrlSelect {
			clr = Hilite
			switch label {
			case SelectHelp:
				text = "HELP"
			case SelectUse:
				text = "USE"
			default:
				text = "SELECT"
			}
		}
		w := TextWidth(text, r.Fonts.Small)
		r.Text(screen, text, r.Fonts.Small,
			float64(rect.Min.X+rect.Dx()/2)-w/2, float64(rect.Min.Y+rect.Dy()/2)-5, clr)
	}
}

// drawCentered вписывает спрайт роли в квадрат со стороной size по центру точки.
func (r *Renderer) drawCentered(screen *ebiten.Image, role string, center image.Point, size float64) {
	img := r.sprites.Frame(role, 0)
	if img == nil {
		return
	}
	w, h := float64(img.Bounds().Dx()), float64(img.Bounds().Dy())
	scale := min(size/w, size/h)
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Scale(scale, scale)
	op.GeoM.Translate(float64(center.X)-w*scale/2, float64(center.Y)-h*scale/2)
	screen.DrawImage(img, op)
}

// drawArrow рисует треугольник направления пиксельными полосками: так он
// попадает в стиль игры и не требует векторных путей.
func (c *Controls) drawArrow(screen *ebiten.Image, dir string, rect image.Rectangle) {
	cx := rect.Min.X + rect.Dx()/2
	cy := rect.Min.Y + rect.Dy()/2
	const steps, step = 5, 2

	for i := 0; i < steps; i++ {
		// Ширина полоски растёт от вершины к основанию.
		half := float32((i + 1) * step)
		var x, y, w, h float32
		switch dir {
		case CtrlUp:
			x, y, w, h = float32(cx)-half, float32(cy-steps+i*2), half*2, step
		case CtrlDown:
			x, y, w, h = float32(cx)-half, float32(cy+steps-i*2-step), half*2, step
		case CtrlLeft:
			x, y, w, h = float32(cx-steps+i*2), float32(cy)-half, step, half*2
		case CtrlRight:
			x, y, w, h = float32(cx+steps-i*2-step), float32(cy)-half, step, half*2
		}
		vector.DrawFilledRect(screen, x, y, w, h, ArrowColor, false)
	}
}

// drawRoundRect и strokeRoundRect — прямоугольники со скруглением.
func drawRoundRect(screen *ebiten.Image, rect image.Rectangle, clr color.Color) {
	vector.DrawFilledRect(screen, float32(rect.Min.X), float32(rect.Min.Y),
		float32(rect.Dx()), float32(rect.Dy()), clr, false)
}

func strokeRoundRect(screen *ebiten.Image, rect image.Rectangle, clr color.Color) {
	vector.StrokeRect(screen, float32(rect.Min.X), float32(rect.Min.Y),
		float32(rect.Dx()), float32(rect.Dy()), 2, clr, false)
}
