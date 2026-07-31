//go:build js

package main

import (
	"syscall/js"

	"github.com/stariydedd/yurnerogue-go/internal/render"
)

// chooseLayout в браузере: на тач-устройствах включается портретная
// «консоль», на десктопе — обычная широкая раскладка.
func chooseLayout() render.Layout {
	window := js.Global().Get("window")
	if !isTouchDevice(window) {
		return render.DesktopLayout()
	}
	w := window.Get("innerWidth").Int()
	h := window.Get("innerHeight").Int()
	return render.TouchLayout(w, h)
}

// isTouchDevice — основной способ ввода тач-экран.
//
// pointer: coarse не срабатывает на ноутбуках с сенсорным экраном, где
// основной ввод всё равно мышь; maxTouchPoints — запасной вариант.
func isTouchDevice(window js.Value) bool {
	if mm := window.Get("matchMedia"); mm.Truthy() {
		if m := window.Call("matchMedia", "(pointer: coarse)"); m.Truthy() {
			if matches := m.Get("matches"); matches.Type() == js.TypeBoolean {
				return matches.Bool()
			}
		}
	}
	if nav := window.Get("navigator"); nav.Truthy() {
		if pts := nav.Get("maxTouchPoints"); pts.Type() == js.TypeNumber {
			return pts.Int() > 0
		}
	}
	return false
}
