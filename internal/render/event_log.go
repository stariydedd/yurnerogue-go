package render

import (
	"image"
	"math"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
	"github.com/stariydedd/yurnerogue-go/internal/domain"
)

type eventLogLine struct {
	text   string
	latest bool
}

func (r *Renderer) eventLogLines(s *domain.Session, width, rows int) []eventLogLine {
	if rows <= 0 || width <= 0 {
		return nil
	}
	messages := s.EventLog
	if len(messages) == 0 && s.Message != "" {
		messages = []string{s.Message}
	}
	var lines []eventLogLine
	for i := len(messages) - 1; i >= 0 && len(lines) < rows; i-- {
		message := strings.TrimSpace(r.translateMessage(messages[i]))
		if message == "" {
			continue
		}
		wrapped := wrapText(message, max(1, width/10))
		limit := min(len(wrapped), 2, rows-len(lines))
		entry := make([]eventLogLine, limit)
		for j := range entry {
			label := wrapped[j]
			if j == limit-1 && len(wrapped) > limit {
				label += "…"
			}
			entry[j] = eventLogLine{fitLabel(label, r.Fonts.Small, float64(width)), i == len(messages)-1}
		}
		lines = append(entry, lines...)
	}
	return lines
}

// eventLogRowHeight is the same on desktop and touch layouts.
const eventLogRowHeight = 16

func (r *Renderer) drawEventLog(dst *ebiten.Image, s *domain.Session, box image.Rectangle) {
	rowHeight := eventLogRowHeight
	const radius = 6
	fillBox(dst, image.Rect(box.Min.X+radius, box.Min.Y, box.Max.X-radius, box.Max.Y), uiInk)
	fillBox(dst, image.Rect(box.Min.X, box.Min.Y+radius, box.Max.X, box.Max.Y-radius), uiInk)
	for _, x := range []int{box.Min.X + radius, box.Max.X - radius} {
		for _, y := range []int{box.Min.Y + radius, box.Max.Y - radius} {
			vector.DrawFilledCircle(dst, float32(x), float32(y), radius, uiInk, false)
		}
	}
	content := eventLogContentBounds(box)
	lines := r.eventLogLines(s, content.Dx(), content.Dy()/rowHeight)
	_, textHeight := text.Measure("Ag", r.Fonts.Small, 0)
	top := eventLogTextTop(box, len(lines), rowHeight, int(math.Ceil(textHeight)))
	for i, line := range lines {
		clr := uiMuted
		if line.latest {
			clr = uiText
		}
		r.Text(dst, line.text, r.Fonts.Small, float64(content.Min.X), float64(top+i*rowHeight), clr)
	}
}

func eventLogTextTop(box image.Rectangle, rows, rowHeight, textHeight int) int {
	// The last line has glyph height only, not a trailing line spacing.
	blockHeight := max(0, rows-1)*rowHeight + textHeight
	return box.Min.Y + (box.Dy()-blockHeight)/2
}

func eventLogContentBounds(frame image.Rectangle) image.Rectangle {
	return image.Rect(frame.Min.X+10, frame.Min.Y+6, frame.Max.X-10, frame.Max.Y-6)
}
