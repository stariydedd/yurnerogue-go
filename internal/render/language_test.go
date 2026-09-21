package render

import (
	"fmt"
	"image"
	"strings"
	"testing"

	"github.com/stariydedd/yurnerogue-go/internal/domain"
	"github.com/stariydedd/yurnerogue-go/internal/locale"
)

func TestLanguageButtonsAreSeparatePointerTargets(t *testing.T) {
	for _, l := range []Layout{DesktopLayout(), TouchLayout(390, 600), TouchLayout(390, 844)} {
		for language, box := range LanguageTargets(l) {
			if !box.In(image.Rect(0, 0, l.ScreenW, l.ScreenH)) || box.Dy() < 44 {
				t.Fatal("language button is outside the screen or too small")
			}
			p := boxCenter(box)
			if LanguageAt(l, p.X, p.Y) != language || MenuActionAt(l, MenuHome, p.X, p.Y) != "" {
				t.Fatal("language flags must have their own pointer-only targets")
			}
			for _, button := range MenuButtons(l, MenuHome) {
				if box.Overlaps(button.Bounds) || button.Bounds.Max.Y > box.Min.Y {
					t.Fatal("language buttons must sit below all other controls")
				}
			}
		}
	}
}

func TestRussianHUDFullLabelsPreserveValues(t *testing.T) {
	fonts, err := loadFonts()
	if err != nil {
		t.Fatal(err)
	}
	l := TouchLayout(390, 600)
	l.Language = locale.Russian
	r := &Renderer{Layout: l, Fonts: fonts}
	p := &domain.Person{Strength: 70, Agility: 70, Health: 500, MaxHealth: 500}
	if got := r.hudStatsLabel(p, fonts.Small, 238, "STRENGTH %d  AGILITY %d"); got != "СИЛА 70  ЛОВКОСТЬ 70" {
		t.Fatalf("full stats should fit: %s", got)
	}
	if r.hudHealthLabel(p, 230) != "ЗДОРОВЬЕ 500 / 500" {
		t.Fatal("health need not be abbreviated")
	}
	p.Strength, p.Agility = 1234, 5678
	if r.hudStatsLabel(p, fonts.Small, 238, "STRENGTH %d  AGILITY %d") != "СИЛ 1234  ЛОВ 5678" {
		t.Fatal("large stats must retain all digits")
	}
	p.Health, p.MaxHealth = 1234567, 7654321
	if r.hudHealthLabel(p, 230) != "ОЗ 1234567 / 7654321" {
		t.Fatal("large health values must retain all digits")
	}
	if TextWidth(fmt.Sprintf(r.tr("LEVEL %d/%d"), 21, 21), fonts.Small) > 130 {
		t.Fatal("full level caption does not fit mobile HUD")
	}
}

func TestEnglishHUDFullLabelsPreserveValues(t *testing.T) {
	fonts, err := loadFonts()
	if err != nil {
		t.Fatal(err)
	}
	for _, l := range []Layout{DesktopLayout(), TouchLayout(390, 600)} {
		r := &Renderer{Layout: l, Fonts: fonts}
		face, width, format := fonts.Compact, 308.0, "STRENGTH %d   AGILITY %d"
		if l.Touch {
			face, width, format = fonts.Small, 238, "STRENGTH %d  AGILITY %d"
		}
		p := &domain.Person{Strength: 70, Agility: 70, Health: 500, MaxHealth: 500}
		if got := r.hudStatsLabel(p, face, width, format); got != fmt.Sprintf(format, 70, 70) || TextWidth(got, face) > width {
			t.Fatalf("full English stats should fit: %s", got)
		}
		if got := r.hudHealthLabel(p, 230); got != "HEALTH 500 / 500" || TextWidth(got, fonts.Small) > 230 {
			t.Fatalf("full English health should fit: %s", got)
		}
		p.Strength, p.Agility, p.Health, p.MaxHealth = 1234, 5678, 1234567, 7654321
		if r.hudStatsLabel(p, face, width, format) != "STR 1234  AGI 5678" || r.hudHealthLabel(p, 230) != "HP 1234567 / 7654321" {
			t.Fatal("large English stats must retain all digits")
		}
	}
}

func TestRussianScreensFitAndRetainNames(t *testing.T) {
	fonts, err := loadFonts()
	if err != nil {
		t.Fatal(err)
	}
	for _, l := range []Layout{DesktopLayout(), TouchLayout(390, 600), TouchLayout(390, 844), TouchLayout(844, 390)} {
		l.Language = locale.Russian
		r := &Renderer{Layout: l, Fonts: fonts}
		rows, height := helpContent(l)
		if height > HelpViewBounds(l).Inset(16).Dy() || HelpScrollLimit(l) != 0 {
			t.Fatalf("Russian help scrolls at %+v", l)
		}
		for _, row := range rows {
			if row.section != "" {
				continue
			}
			if strings.Join(row.lines, " ") != r.tr(row.entry.desc) {
				t.Fatal("translated help lost words")
			}
		}
		for i, entry := range helpItems {
			if helpEntryName(l, entry) != []string{"ЕДА", "ЗЕЛЬЯ", "СВИТКИ", "ОРУЖИЕ", "ПОРТАЛ"}[i] {
				t.Fatal("help must use category names")
			}
		}
		for _, entry := range helpEnemies {
			if helpEntryName(l, entry) != entry.name {
				t.Fatal("hero name changed")
			}
		}
		boxHelp := HUDTargets(l)[CtrlSelect]
		if l.Touch {
			boxHelp = NewControls(l).targets[CtrlSelect]
		}
		if TextWidth(r.tr("HELP"), fonts.Small) > float64(boxHelp.Dx()-8) {
			t.Fatal("help button clips Russian caption")
		}
		for _, page := range []MenuPage{MenuHome, MenuPause, MenuWelcome, MenuResults, MenuQuit, MenuHelp} {
			for _, button := range MenuButtons(l, page) {
				face := fonts.Menu
				if page == MenuHome && (button.Action == "music" || button.Action == "effects") {
					face = fonts.UI
				}
				if TextWidth(r.tr(button.Label), face) > float64(button.Bounds.Dx()-16) {
					t.Fatalf("Russian button clipped: %s", button.Label)
				}
			}
		}
		for key, box := range AudioTargets(l) {
			label := "SFX %d%%"
			if key == "music" {
				label = "MUSIC %d%%"
			}
			if TextWidth(fmt.Sprintf(r.tr(label), 100), fonts.UI) > float64(box.Dx()-16) {
				t.Fatal("volume caption clipped")
			}
		}
		box := runPanelBounds(l, false)
		welcome := r.welcomeText(box)
		for _, row := range welcome.rows {
			if row.y+row.height > box.Max.Y-16 || TextWidth(r.tr(row.entry.name), fonts.Menu) > float64(box.Dx()-88) {
				t.Fatalf("Russian welcome does not fit: %s", row.entry.name)
			}
		}
		resultBox := runPanelBounds(l, true)
		face := r.secondaryFace()
		chars := int(float64(resultBox.Dx()) / TextWidth("M", face))
		for _, status := range []string{
			"Submitting score...", "Score submitted to global leaderboard!",
			"Score not submitted: leaderboard unavailable for this game.",
			"Game length limit reached: score cannot be submitted.",
			"Score submission could not be confirmed.",
			"New game: Connection timed out. Press PLAY to retry.",
		} {
			lines := wrapText(r.translateMessage(status), chars)
			if resultBox.Max.Y+16+len(lines)*(int(TextWidth("M", face))+6) > MenuButtons(l, MenuResults)[0].Bounds.Min.Y {
				t.Fatalf("Russian status overlaps buttons: %s", status)
			}
		}
	}
}
