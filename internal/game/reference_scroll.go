package game

import (
	"image"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/stariydedd/yurnerogue-go/internal/render"
)

func (g *Game) referenceScrollState() (image.Rectangle, *int, int, bool) {
	if g.renderer == nil {
		return image.Rectangle{}, nil, 0, false
	}
	switch g.state {
	case StateHelp:
		return render.HelpViewBounds(g.renderer.Layout), &g.helpScroll, render.HelpScrollLimit(g.renderer.Layout), true
	}
	return image.Rectangle{}, nil, 0, false
}

func (g *Game) scrollReference(delta int) {
	_, offset, limit, ok := g.referenceScrollState()
	if ok {
		*offset = max(0, min(limit, *offset+delta))
	}
}

func (g *Game) handleReferenceKey(key ebiten.Key) bool {
	view, offset, limit, ok := g.referenceScrollState()
	if !ok {
		return false
	}
	switch key {
	case ebiten.KeyUp, ebiten.KeyW:
		g.scrollReference(-48)
	case ebiten.KeyDown, ebiten.KeyS:
		g.scrollReference(48)
	case ebiten.KeyPageUp:
		g.scrollReference(-view.Dy() * 3 / 4)
	case ebiten.KeyPageDown:
		g.scrollReference(view.Dy() * 3 / 4)
	case ebiten.KeyHome:
		*offset = 0
	case ebiten.KeyEnd:
		*offset = limit
	default:
		return false
	}
	return true
}

func (g *Game) beginReferenceDrag(id ebiten.TouchID, x, y int) bool {
	view, _, limit, ok := g.referenceScrollState()
	if !ok || limit == 0 || !image.Pt(x, y).In(view) {
		return false
	}
	g.referenceDragging, g.referenceDragID, g.referenceDragY = true, id, y
	return true
}

func (g *Game) updateReferenceScroll() bool {
	view, _, _, ok := g.referenceScrollState()
	if !ok || !ebiten.IsFocused() {
		g.referenceDragging = false
		return false
	}
	if g.referenceDragging {
		var x, y int
		if g.referenceDragID == mouseID {
			if !ebiten.IsMouseButtonPressed(ebiten.MouseButtonLeft) {
				g.referenceDragging = false
				return true
			}
			x, y = ebiten.CursorPosition()
		} else {
			active := false
			for _, id := range ebiten.AppendTouchIDs(nil) {
				active = active || id == g.referenceDragID
			}
			if !active {
				g.referenceDragging = false
				return true
			}
			x, y = ebiten.TouchPosition(g.referenceDragID)
		}
		_, y = g.toLogical(x, y)
		g.scrollReference(g.referenceDragY - y)
		g.referenceDragY = y
		return true
	}
	x, y := g.toLogical(ebiten.CursorPosition())
	if _, wheel := ebiten.Wheel(); wheel != 0 && image.Pt(x, y).In(view) {
		g.scrollReference(int(math.Round(-wheel * 48)))
	}
	if ids := inpututil.AppendJustPressedTouchIDs(nil); len(ids) > 0 {
		x, y = g.toLogical(ebiten.TouchPosition(ids[0]))
		return g.beginReferenceDrag(ids[0], x, y)
	}
	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		return g.beginReferenceDrag(mouseID, x, y)
	}
	return false
}
