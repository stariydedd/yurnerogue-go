//go:build !js

package main

import (
	"os"

	"github.com/stariydedd/yurnerogue-go/internal/render"
)

// chooseLayout вне браузера: обычная раскладка с клавиатурой.
// ROGUE_TOUCH=1 форсит портретную «консоль» — удобно отлаживать её
// в нативном окне, не поднимая стенд для телефона.
func chooseLayout() render.Layout {
	if os.Getenv("ROGUE_TOUCH") == "1" {
		return render.TouchLayout(390, 844)
	}
	return render.DesktopLayout()
}
