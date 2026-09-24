package render

import (
	"math/rand"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/stariydedd/yurnerogue-go/internal/domain"
	"github.com/stariydedd/yurnerogue-go/internal/locale"
)

func TestGeneratedItemNamesFitHUDAndMobileMenus(t *testing.T) {
	fonts, err := loadFonts()
	if err != nil {
		t.Fatal(err)
	}
	p := domain.NewSessionSeed(21).Player
	p.MaxHealth, p.Strength, p.Agility = 1000000, 1000000, 1000000
	rng := rand.New(rand.NewSource(21))
	for i := 0; i < 2000; i++ {
		for _, item := range []*domain.Item{domain.NewFood(p, domain.MaxLevels), domain.NewElixir(p, domain.MaxLevels), domain.NewScroll(p, domain.MaxLevels), domain.NewWeapon(domain.MaxLevels, rng)} {
			label := "> " + item.Name + item.StatLabel()
			if TextWidth(label, fonts.Compact) > 460 {
				t.Fatalf("mobile menu clips %q", label)
			}
			if item.Type == domain.ItemWeapon {
				p.Weapon = item
				if TextWidth(weaponLabel(p), fonts.Small) > 238 || TextWidth(weaponLabel(p), fonts.Compact) > 244 {
					t.Fatalf("HUD clips %q", weaponLabel(p))
				}
			}
		}
	}
}

func TestHUDItemValuesFollowInventoryAndEquipment(t *testing.T) {
	p := domain.NewPerson()
	for _, control := range []string{CtrlFood, CtrlElixir} {
		if value, active := itemSlotValue(p, control); value != "0" || active {
			t.Fatalf("empty %s: %q %v", control, value, active)
		}
	}
	food := &domain.Item{Type: domain.ItemFood}
	p.PickUpItem(food)
	p.PickUpItem(&domain.Item{Type: domain.ItemFood})
	if value, active := itemSlotValue(p, CtrlFood); value != "2" || !active {
		t.Fatal("food count incorrect")
	}
	p.UseItem(food)
	if value, _ := itemSlotValue(p, CtrlFood); value != "1" {
		t.Fatal("used item still counted")
	}
	if weaponLabel(p) != domain.BaseWeaponName {
		t.Fatal("wrong starting weapon label")
	}
	p.Weapon = &domain.Item{Type: domain.ItemWeapon, Name: "Yasha", StrengthEffect: 40}
	if weaponLabel(p) != "Yasha" {
		t.Fatal("the weapon name alone shows its strength")
	}
	if value, active := itemSlotValue(nil, CtrlFood); value != "" || active {
		t.Fatal("menu without session has inventory")
	}
}

func TestHealthBarTurnsRedStrictlyBelowQuarterHealth(t *testing.T) {
	for _, test := range []struct {
		health, maximum int
		red             bool
	}{
		{500, 500, false}, {126, 500, false}, {125, 500, false}, {124, 500, true},
		{0, 500, true}, {25, 101, true}, {26, 101, false}, {1, 4, false},
		{0, 0, false}, {600, 500, false},
	} {
		p := &domain.Person{Health: test.health, MaxHealth: test.maximum}
		fill, highlight, edge := healthBarColors(p)
		if (fill == uiDanger) != test.red || test.red && (edge != uiDanger || highlight == uiHighlight) {
			t.Fatalf("incorrect health bar palette at %d/%d", test.health, test.maximum)
		}
	}
	p := &domain.Person{Health: 100, MaxHealth: 500}
	p.Heal(25)
	if fill, _, _ := healthBarColors(p); fill != uiAccent {
		t.Fatal("healing back to 25 percent must restore orange")
	}
	p.MaxHealth = 600
	if fill, _, _ := healthBarColors(p); fill != uiDanger {
		t.Fatal("health warning must use the current maximum health")
	}
}

func TestHUDEffectTimerFollowsTurnsAndSleep(t *testing.T) {
	p := domain.NewPerson()
	if len(statusBonuses(p)) != 0 {
		t.Fatal("phantom effect")
	}
	add := func(item *domain.Item) { p.PickUpItem(item); p.UseItem(item) }
	add(&domain.Item{Type: domain.ItemElixir, StrengthEffect: 3})
	p.TickEffects()
	add(&domain.Item{Type: domain.ItemElixir, AgilityEffect: 5})
	add(&domain.Item{Type: domain.ItemElixir, MaxHealthEffect: 20})
	bonuses := statusBonuses(p)
	if len(bonuses) != 3 || bonusLabel(bonuses[0], true) != "STR +3 19T" || bonusLabel(bonuses[1], true) != "AGI +5 20T" || bonusLabel(bonuses[2], true) != "MHP +20 20T" {
		t.Fatalf("not all stat bonuses visible: %+v", bonuses)
	}
	p.FallAsleep(2)
	if len(statusBonuses(p)) != 3 {
		t.Fatal("sleep hid potion bonuses")
	}
	p.TickSleep()
	p.TickSleep()
	for i := 0; i < domain.ElixirDuration; i++ {
		p.TickEffects()
	}
	if len(statusBonuses(p)) != 0 {
		t.Fatal("expired effect still visible")
	}
}

func TestHUDStackedBonusesUpdateAtEachExpiry(t *testing.T) {
	p := domain.NewPerson()
	add := func(amount int) {
		item := &domain.Item{Type: domain.ItemElixir, StrengthEffect: amount}
		p.PickUpItem(item)
		p.UseItem(item)
	}
	add(3)
	for i := 0; i < 7; i++ {
		p.TickEffects()
	}
	add(5)
	bonuses := statusBonuses(p)
	if len(bonuses) != 1 || bonuses[0].Amount != 8 || bonuses[0].TurnsLeft != 13 {
		t.Fatalf("incorrect stack: %+v", bonuses)
	}
	for i := 0; i < 13; i++ {
		p.TickEffects()
	}
	bonuses = statusBonuses(p)
	if len(bonuses) != 1 || bonuses[0].Amount != 5 || bonuses[0].TurnsLeft != 7 {
		t.Fatalf("expired bonus still counted: %+v", bonuses)
	}
}

func TestSleepSharesStatusAreaWithAllBonuses(t *testing.T) {
	p := domain.NewPerson()
	for _, item := range []*domain.Item{
		{Type: domain.ItemElixir, StrengthEffect: 3},
		{Type: domain.ItemElixir, AgilityEffect: 5},
		{Type: domain.ItemElixir, MaxHealthEffect: 20},
	} {
		p.PickUpItem(item)
		p.UseItem(item)
	}
	p.FallAsleep(2)
	for _, layout := range []Layout{DesktopLayout(), TouchLayout(390, 844)} {
		for _, language := range []locale.Language{locale.English, locale.Russian} {
			layout.Language = language
			r := &Renderer{Layout: layout}
			effects := r.hudEffects(p)
			if len(effects) != 4 || effects[0].tint != uiDebuff {
				t.Fatalf("sleep must use the blue debuff color: %+v", effects)
			}
			for _, effect := range effects[1:] {
				if effect.tint != uiAccent {
					t.Fatal("potion buffs must keep their orange color")
				}
			}
			wantHead, wantTurns := "SLEEP", "2T"
			if language == locale.Russian {
				wantHead, wantTurns = "СОН", "2Х"
			}
			if len(effects) != 4 || effects[0].head != wantHead || effects[0].turns != wantTurns {
				t.Fatalf("missing sleep or potion statuses: %+v", effects)
			}
		}
	}
	p.TickSleep()
	p.TickSleep()
	if effects := (&Renderer{Layout: DesktopLayout()}).hudEffects(p); len(effects) != 3 {
		t.Fatalf("sleep must disappear on waking: %+v", effects)
	}
}

func TestHUDAllBonusLabelsFitMobileColumn(t *testing.T) {
	fonts, err := loadFonts()
	if err != nil {
		t.Fatal(err)
	}
	for _, stat := range []domain.ItemSubType{domain.SubStrength, domain.SubAgility, domain.SubHealth} {
		label := bonusLabel(domain.EffectStatus{Stat: stat, Amount: 999, TurnsLeft: 20}, true)
		if TextWidth(label, fonts.Small) > 130 {
			t.Fatalf("mobile bonus clipped: %s", label)
		}
	}
}

func TestBonusStyleIsSharedAndFitsDesktopRow(t *testing.T) {
	fonts, err := loadFonts()
	if err != nil {
		t.Fatal(err)
	}
	r := &Renderer{Fonts: fonts, Layout: DesktopLayout()}
	bonuses := []domain.EffectStatus{
		{Stat: domain.SubStrength, Amount: 3, TurnsLeft: 20},
		{Stat: domain.SubAgility, Amount: 5, TurnsLeft: 20},
		{Stat: domain.SubHealth, Amount: 20, TurnsLeft: 20},
	}
	for language, want := range map[locale.Language][2]string{
		locale.English: {"STR +3", "20T"},
		locale.Russian: {"СИЛ +3", "20Х"},
	} {
		r.Layout.Language = language
		if head, turns := r.bonusParts(bonuses[0]); head != want[0] || turns != want[1] {
			t.Fatalf("%s bonus parts: %q %q", language, head, turns)
		}
		width := -10.0
		for _, bonus := range bonuses {
			head, turns := r.bonusParts(bonus)
			width += TextWidth(head+" "+turns, fonts.Small) + 10
		}
		if room := desktopHUDGeometry(r.Layout).effects.Dx(); width > float64(room) {
			t.Fatalf("%s: typical bonuses need %.0fpx, row has %dpx", language, width, room)
		}
	}
}

func TestHUDLabelsStayWithinTheirColumns(t *testing.T) {
	fonts, err := loadFonts()
	if err != nil {
		t.Fatal(err)
	}
	for _, label := range []string{"Juggernaut Blade +99999", strings.Repeat("long-word-", 20), "Очень длинное название предмета", "HP 999999 / 999999"} {
		for _, width := range []float64{0, 10, 130, 238, 244} {
			got := fitLabel(label, fonts.Small, width)
			if !utf8.ValidString(got) || TextWidth(got, fonts.Small) > width {
				t.Fatalf("label %q exceeds %v", got, width)
			}
		}
	}
	if got := fitLabel("Tango", fonts.Small, 130); got != "Tango" {
		t.Fatalf("short label changed: %q", got)
	}
}

func TestWrapTextKeepsAllWords(t *testing.T) {
	s := "Moves 2 tiles. Rests, counters, never misses."
	got := wrapText(s, 20)

	if strings.Join(got, " ") != s {
		t.Fatalf("перенос исказил текст: %q", got)
	}
	for i, line := range got {
		if len(line) > 20 {
			t.Fatalf("строка %d длиннее лимита: %q", i, line)
		}
	}
}

func TestWrapTextHandlesOverlongWord(t *testing.T) {
	// Слово длиннее лимита обрезать нельзя — иначе текст потеряется;
	// оно занимает строку целиком.
	got := wrapText("Parchment-of-Thunderous-Roar tail", 10)
	if len(got) == 0 || got[0] != "Parchment-of-Thunderous-Roar" {
		t.Fatalf("длинное слово должно остаться целым, получено %q", got)
	}
}
