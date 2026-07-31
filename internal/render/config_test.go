package render

import "testing"

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
