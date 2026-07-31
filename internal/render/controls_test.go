package render

import (
	"image"
	"testing"
)

// touchControls — панель кнопок для типового телефона.
func touchControls() (Layout, *Controls) {
	l := TouchLayout(390, 844)
	return l, NewControls(l)
}

func TestControlsFitInsidePanel(t *testing.T) {
	l, c := touchControls()

	if c.Panel.Max.Y > l.ScreenH {
		t.Fatalf("панель вылезает за экран: %d > %d", c.Panel.Max.Y, l.ScreenH)
	}
	for name, rect := range c.dpad {
		if !rect.In(c.Panel) {
			t.Fatalf("клавиша %q вне панели: %v", name, rect)
		}
	}
	for name, rect := range c.pills {
		if !rect.In(c.Panel) {
			t.Fatalf("кнопка %q вне панели: %v", name, rect)
		}
	}
	for name, center := range c.buttons {
		box := image.Rect(center.X-c.btnRadius, center.Y-c.btnRadius,
			center.X+c.btnRadius, center.Y+c.btnRadius)
		if !box.In(c.Panel) {
			t.Fatalf("кнопка предмета %q вне панели: %v", name, box)
		}
	}
}

func TestControlAtHitsEveryButton(t *testing.T) {
	_, c := touchControls()

	// Крестовина: центр каждой клавиши отдаёт своё направление.
	for _, name := range []string{CtrlUp, CtrlDown, CtrlLeft, CtrlRight} {
		rect := c.dpad[name]
		cx, cy := rect.Min.X+rect.Dx()/2, rect.Min.Y+rect.Dy()/2
		if got := c.ControlAt(cx, cy); got != name {
			t.Fatalf("в центре %q попали в %q", name, got)
		}
	}

	if got := c.ControlAt(c.dpadCent.X, c.dpadCent.Y); got != CtrlRun {
		t.Fatalf("центр крестовины должен быть бегом, получено %q", got)
	}

	for _, name := range []string{CtrlWeapon, CtrlFood, CtrlElixir, CtrlScroll} {
		center := c.buttons[name]
		if got := c.ControlAt(center.X, center.Y); got != name {
			t.Fatalf("в центре кнопки %q попали в %q", name, got)
		}
	}

	for _, name := range []string{CtrlMenu, CtrlSelect} {
		rect := c.pills[name]
		cx, cy := rect.Min.X+rect.Dx()/2, rect.Min.Y+rect.Dy()/2
		if got := c.ControlAt(cx, cy); got != name {
			t.Fatalf("в центре %q попали в %q", name, got)
		}
	}
}

func TestControlAtIgnoresEmptySpace(t *testing.T) {
	l, c := touchControls()

	// Карта и статус-панель кнопок не содержат.
	for _, p := range []image.Point{
		{X: l.ScreenW / 2, Y: 10},
		{X: l.ScreenW / 2, Y: l.GridH + 10},
	} {
		if got := c.ControlAt(p.X, p.Y); got != "" {
			t.Fatalf("в точке %v нашли кнопку %q, хотя там карта или HUD", p, got)
		}
	}
	// Промежуток между крестовиной и ромбом предметов.
	midX := (c.dpadCent.X + c.buttons[CtrlFood].X) / 2
	if got := c.ControlAt(midX, c.dpadCent.Y); got != "" {
		t.Fatalf("между блоками кнопок нашли %q", got)
	}
}

func TestDPadAndItemButtonsDoNotOverlap(t *testing.T) {
	// Иначе палец на крестовине случайно съедал бы предмет.
	_, c := touchControls()

	for dname, rect := range c.dpad {
		for bname, center := range c.buttons {
			box := image.Rect(center.X-c.btnRadius, center.Y-c.btnRadius,
				center.X+c.btnRadius, center.Y+c.btnRadius)
			if rect.Overlaps(box) {
				t.Fatalf("клавиша %q пересекается с кнопкой %q", dname, bname)
			}
		}
	}
}
