package render

import (
	"math/rand"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/stariydedd/yurnerogue-go/internal/domain"
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
		for _, item := range []*domain.Item{domain.NewFood(p), domain.NewElixir(p), domain.NewScroll(p), domain.NewWeapon(rng)} {
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
	for _, control := range []string{CtrlFood, CtrlElixir, CtrlScroll} {
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
	if value, active := itemSlotValue(p, CtrlWeapon); value != "+0" || !active {
		t.Fatal("starter weapon missing from HUD")
	}
	if weaponLabel(p) != domain.BaseWeaponName {
		t.Fatal("wrong starting weapon label")
	}
	p.Weapon = &domain.Item{Type: domain.ItemWeapon, StrengthEffect: 12}
	if value, active := itemSlotValue(p, CtrlWeapon); value != "+12" || !active {
		t.Fatal("wrong weapon bonus")
	}
	if value, active := itemSlotValue(nil, CtrlFood); value != "" || active {
		t.Fatal("menu without session has inventory")
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

func TestMessageLinesKeepsShortMessageWhole(t *testing.T) {
	got := messageLines("You missed the Pudge.", 40)
	if len(got) != 1 || got[0] != "You missed the Pudge." {
		t.Fatalf("короткое сообщение не должно разбиваться, получено %q", got)
	}
}

func TestMessageLinesSplitsAtColon(t *testing.T) {
	// «Picked up: имя предмета» ломается по двоеточию: заголовок отдельно,
	// длинное имя отдельно — так оно помещается на узком экране.
	msg := "Picked up: Elixir of the Phantom's Breath [+3 STR]."
	got := messageLines(msg, 44)

	if len(got) != 2 {
		t.Fatalf("ожидалось две строки, получено %d: %q", len(got), got)
	}
	if got[0] != "Picked up:" {
		t.Fatalf("первая строка %q, ожидалось «Picked up:»", got[0])
	}
	if !strings.HasPrefix(got[1], "Elixir") {
		t.Fatalf("вторая строка должна начинаться с названия, получено %q", got[1])
	}
	for i, line := range got {
		if len(line) > 44 {
			t.Fatalf("строка %d длиннее лимита: %q", i, line)
		}
	}
}

func TestMessageLinesWrapsWithoutColon(t *testing.T) {
	msg := "The Skywrath Mage hit you for 42 dmg. You fall asleep!"
	got := messageLines(msg, 30)

	if len(got) < 2 {
		t.Fatalf("длинное сообщение должно переноситься, получено %q", got)
	}
	for i, line := range got {
		if len(line) > 30 {
			t.Fatalf("строка %d длиннее лимита: %q", i, line)
		}
	}
	if strings.Join(got, " ") != msg {
		t.Fatalf("перенос потерял или переставил слова: %q", got)
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
