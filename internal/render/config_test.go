package render

import (
	"image"
	"math"
	"testing"
)

func TestAdaptiveDesktopFitsViewportAndKeepsHUDReadable(t *testing.T) {
	for _, size := range []image.Point{{1280, 848}, {1280, 720}, {1366, 768}, {1920, 1080}, {2560, 1440}, {3440, 1440}, {3840, 2160}, {1024, 768}, {800, 600}} {
		l := DesktopLayoutForSize(size.X, size.Y)
		if l.ScreenW < 1280 || l.ScreenH < 832 || l.Touch || l.PanelH != 128 || l.ControlsH != 0 {
			t.Fatalf("unreadable desktop layout at %v: %+v", size, l)
		}
		if l.GridW != l.ScreenW || l.GridH+l.PanelH != l.ScreenH {
			t.Fatalf("viewport does not fill the layout at %v", size)
		}
		scale := math.Min(float64(size.X)/float64(l.ScreenW), float64(size.Y)/float64(l.ScreenH))
		if float64(size.X)-float64(l.ScreenW)*scale > scale+0.001 || float64(size.Y)-float64(l.ScreenH)*scale > scale+0.001 {
			t.Fatalf("desktop leaves more than a rounding margin at %v", size)
		}
		targets := HUDTargets(l)
		for name, box := range targets {
			if !box.In(image.Rect(0, l.GridH, l.ScreenW, l.ScreenH)) {
				t.Fatalf("HUD %s clipped at %v", name, size)
			}
		}
		for _, page := range []MenuPage{MenuHome, MenuPause, MenuWelcome, MenuResults, MenuQuit, MenuHelp, MenuLeaderboard, MenuName} {
			for _, button := range MenuButtons(l, page) {
				if !button.Bounds.In(image.Rect(0, 0, l.ScreenW, l.ScreenH)) {
					t.Fatalf("menu button %s clipped at %v", button.Label, size)
				}
			}
		}
	}
	if DesktopLayoutForSize(1280, 832) != DesktopLayout() || DesktopLayoutForSize(0, 0) != DesktopLayout() {
		t.Fatal("reference layout or invalid-size fallback changed")
	}
}

func TestDesktopHUDUsesFullWidthWithoutOverlap(t *testing.T) {
	for _, size := range []image.Point{{1280, 848}, {1920, 1080}, {3440, 1440}, {3840, 2160}, {800, 600}} {
		l := DesktopLayoutForSize(size.X, size.Y)
		h := desktopHUDGeometry(l)
		panel := image.Rect(0, l.GridTop(), l.ScreenW, l.ControlsTop())
		if h.portrait.Min.X != 16 || h.targets[CtrlMenu].Max.X != l.ScreenW-16 {
			t.Fatalf("HUD is not anchored to both edges at %v", size)
		}
		if h.health.Dx() < 302 || h.status.Dx() < 172 {
			t.Fatalf("HUD text area shrank at %v", size)
		}
		regions := []image.Rectangle{h.portrait, h.health, h.effects, h.status, h.log}
		for _, rect := range h.targets {
			regions = append(regions, rect)
		}
		for i, rect := range regions {
			if !rect.In(panel) {
				t.Fatalf("HUD region outside panel at %v: %v", size, rect)
			}
			for _, other := range regions[i+1:] {
				if rect.Overlaps(other) {
					t.Fatalf("HUD regions overlap at %v: %v and %v", size, rect, other)
				}
			}
		}
		previousRight := h.inventory.Min.X - 1
		for _, name := range []string{CtrlStrike, CtrlGuard, CtrlFood, CtrlElixir} {
			rect := h.targets[name]
			if rect.Min.X <= previousRight || rect.Dx() != 72 || rect.Dy() != 72 || !rect.In(h.inventory) {
				t.Fatalf("inventory row is inconsistent at %v: %s", size, name)
			}
			previousRight = rect.Max.X
			center := boxCenter(rect)
			if HUDControlAt(l, center.X, center.Y) != name {
				t.Fatalf("inventory hit target detached at %v: %s", size, name)
			}
		}
	}
}

func TestDesktopLayoutAddsUp(t *testing.T) {
	l := DesktopLayout()

	if l.Touch {
		t.Fatal("десктопная раскладка не должна быть тач")
	}
	if l.ControlsH != 0 {
		t.Fatalf("панель кнопок на десктопе не нужна, получено %d", l.ControlsH)
	}
	if got := l.GridH + l.PanelH; got != l.ScreenH {
		t.Fatalf("поле и панель дают %d, а экран %d", got, l.ScreenH)
	}
	if l.GridW != l.ScreenW {
		t.Fatalf("поле шириной %d при экране %d", l.GridW, l.ScreenW)
	}
	if l.GridW != ViewCols*TileSize || l.GridH != ViewRows*TileSize {
		t.Fatalf("поле %dx%d не кратно тайлам", l.GridW, l.GridH)
	}
}

func TestTouchLayoutAddsUp(t *testing.T) {
	// Высоты трёх полос обязаны в сумме давать экран, иначе панель кнопок
	// уедет за край или между блоками появится щель.
	sizes := [][2]int{{390, 844}, {360, 640}, {430, 932}, {768, 1024}}
	for _, s := range sizes {
		l := TouchLayout(s[0], s[1])
		if !l.Touch {
			t.Fatalf("%v: раскладка должна быть тач", s)
		}
		if got := l.GridH + l.PanelH + l.ControlsH; got != l.ScreenH {
			t.Fatalf("%v: сумма полос %d, экран %d", s, got, l.ScreenH)
		}
		if l.GridH <= 0 {
			t.Fatalf("%v: на карту не осталось места (%d)", s, l.GridH)
		}
		if l.ControlsTop() != l.GridH+l.PanelH {
			t.Fatalf("%v: панель кнопок начинается не там, где кончается статус", s)
		}
		if l.GridTop() != l.GridH {
			t.Fatalf("%v: статус-панель начинается не там, где кончается карта", s)
		}
	}
}

func TestTouchLayoutClampsProportions(t *testing.T) {
	// Пропорции окна ограничены: на очень широком или очень длинном экране
	// раскладка не должна вырождаться.
	wide := TouchLayout(1000, 500)   // соотношение 0.5
	tall := TouchLayout(300, 1500)   // соотношение 5.0
	square := TouchLayout(500, 1000) // соотношение 2.0 — внутри диапазона

	if wide.ScreenH != int(float64(wide.ScreenW)*1.6) {
		t.Fatalf("широкое окно должно упираться в нижнюю границу 1.6, получено %d/%d",
			wide.ScreenH, wide.ScreenW)
	}
	if tall.ScreenH != int(float64(tall.ScreenW)*2.3) {
		t.Fatalf("длинное окно должно упираться в верхнюю границу 2.3, получено %d/%d",
			tall.ScreenH, tall.ScreenW)
	}
	if square.ScreenH != square.ScreenW*2 {
		t.Fatalf("соотношение внутри диапазона должно сохраняться, получено %d/%d",
			square.ScreenH, square.ScreenW)
	}
}
