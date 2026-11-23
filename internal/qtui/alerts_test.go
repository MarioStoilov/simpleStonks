package qtui

import (
	"testing"

	"github.com/MarioStoilov/simplestonks/internal/notify"
)

func TestParseAlertPrice(t *testing.T) {
	valid := map[string]float64{
		"110":      110,
		"110.5":    110.5,
		"110.55":   110.55,
		"0.5":      0.5,
		" 42.00 ":  42,
		"00123.45": 123.45,
	}
	for input, want := range valid {
		value, ok := parseAlertPrice(input)
		if !ok || value != want {
			t.Errorf("parseAlertPrice(%q) = %v/%v, want %v/true", input, value, ok, want)
		}
	}

	invalid := []string{
		"", "0", "0.00", "-5", "+5", "110.555", "110.", ".5", "abc",
		"1e3", "110,50", "110.5.5", "11 0",
	}
	for _, input := range invalid {
		if value, ok := parseAlertPrice(input); ok {
			t.Errorf("parseAlertPrice(%q) = %v/true, want invalid", input, value)
		}
	}
}

func TestAlertPillText(t *testing.T) {
	above := notify.Alert{Symbol: "AAPL", Price: 250, Above: true}
	below := notify.Alert{Symbol: "AAPL", Price: 180, Above: false}
	cases := []struct {
		name    string
		alert   notify.Alert
		current float64
		known   bool
		want    string
	}{
		{"above the price", above, 240, true, "▲ 250.00 (+4.17%)"},
		{"below the price", below, 200, true, "▼ 180.00 (-10.00%)"},
		{"price moved past an above-alert", above, 260, true, "▲ 250.00 (-3.85%)"},
		{"at the price", above, 250, true, "▲ 250.00 (+0.00%)"},
		{"no price yet", above, 0, false, "▲ 250.00"},
		{"unusable price", below, 0, true, "▼ 180.00"},
	}
	for _, testCase := range cases {
		if got := alertPillText(testCase.alert, testCase.current, testCase.known); got != testCase.want {
			t.Errorf("%s: alertPillText = %q, want %q", testCase.name, got, testCase.want)
		}
	}
}

func TestParseAlertPercent(t *testing.T) {
	valid := map[string]float64{
		"10":      10,
		"+10":     10,
		"-10":     -10,
		"-10%":    -10,
		" 2.5 % ": 2.5,
		"-99.99":  -99.99,
		"0":       0,
		"250":     250,
	}
	for input, want := range valid {
		value, ok := parseAlertPercent(input)
		if !ok || value != want {
			t.Errorf("parseAlertPercent(%q) = %v/%v, want %v/true", input, value, ok, want)
		}
	}

	invalid := []string{"", "-", "+", "%", "-100", "-150", "1.234", "--5", "+-5", "5%%", "abc", "1e2", "5,5"}
	for _, input := range invalid {
		if value, ok := parseAlertPercent(input); ok {
			t.Errorf("parseAlertPercent(%q) = %v/true, want invalid", input, value)
		}
	}
}

// TestAlertEntryPriceAnchor: an entered price keeps its value while its
// percentage follows the current price.
func TestAlertEntryPriceAnchor(t *testing.T) {
	entry := &alertEntry{current: 200}
	entry.editPrice("220")
	if entry.percentText != "+10.00" || entry.direction() != directionUp {
		t.Errorf("after entering 220 at 200: percent %q, direction %q", entry.percentText, entry.direction())
	}
	entry.setCurrent(250)
	if entry.priceText != "220" || entry.percentText != "-12.00" || entry.direction() != directionDown {
		t.Errorf("after the price moved to 250: price %q, percent %q, direction %q",
			entry.priceText, entry.percentText, entry.direction())
	}
	entry.editPrice("22x")
	if entry.percentText != "" || entry.direction() != directionFlat {
		t.Errorf("an invalid price left percent %q, direction %q", entry.percentText, entry.direction())
	}
	if _, ok := entry.price(); ok {
		t.Error("an invalid price was accepted")
	}
}

// TestAlertEntryPercentAnchor: an entered percentage keeps its value while
// its price follows the current price.
func TestAlertEntryPercentAnchor(t *testing.T) {
	entry := &alertEntry{current: 200}
	entry.editPercent("-10")
	if entry.priceText != "180.00" || entry.direction() != directionDown {
		t.Errorf("after entering -10%% at 200: price %q, direction %q", entry.priceText, entry.direction())
	}
	entry.setCurrent(150)
	value, ok := entry.price()
	if entry.percentText != "-10" || entry.priceText != "135.00" || !ok || value != 135 {
		t.Errorf("after the price moved to 150: percent %q, price %q (%v/%v)",
			entry.percentText, entry.priceText, value, ok)
	}
	entry.editPercent("-100")
	if entry.priceText != "" {
		t.Errorf("an impossible percentage left price %q", entry.priceText)
	}
	// Switching back to the price field re-anchors on it.
	entry.editPrice("165")
	entry.setCurrent(100)
	if entry.priceText != "165" || entry.percentText != "+65.00" {
		t.Errorf("after re-anchoring on the price: price %q, percent %q", entry.priceText, entry.percentText)
	}
}

// TestAlertEntryFocusSwitchesTheAnchor: entering the derived field freezes
// its value without touching either text; from then on the other field
// follows.
func TestAlertEntryFocusSwitchesTheAnchor(t *testing.T) {
	entry := &alertEntry{current: 200}
	entry.editPrice("123.45")
	if entry.percentText != "-38.27" {
		t.Fatalf("percent = %q, want -38.27", entry.percentText)
	}
	entry.focus(anchorPercent)
	if entry.priceText != "123.45" || entry.percentText != "-38.27" {
		t.Errorf("entering the percentage changed the fields: price %q, percent %q",
			entry.priceText, entry.percentText)
	}
	entry.setCurrent(300)
	if entry.percentText != "-38.27" || entry.priceText != "185.19" {
		t.Errorf("after a tick with the percentage active: price %q, percent %q",
			entry.priceText, entry.percentText)
	}
	entry.focus(anchorPrice)
	entry.setCurrent(370.38)
	if entry.priceText != "185.19" || entry.percentText != "-50.00" {
		t.Errorf("after a tick with the price active again: price %q, percent %q",
			entry.priceText, entry.percentText)
	}
}

// TestAlertEntryEmptyAnchorKeepsTheOtherField: entering an empty field must
// not wipe the value in the other one on the next tick.
func TestAlertEntryEmptyAnchorKeepsTheOtherField(t *testing.T) {
	entry := &alertEntry{current: 200}
	entry.priceText = "220" // a price in place, the percentage still empty
	entry.focus(anchorPercent)
	entry.setCurrent(210)
	if value, ok := entry.price(); !ok || value != 220 {
		t.Errorf("price after entering the empty percentage = %v/%v, want 220/true", value, ok)
	}
}

// TestAlertEntryWithoutEdits: price ticks before any input leave both
// fields empty.
func TestAlertEntryWithoutEdits(t *testing.T) {
	entry := &alertEntry{current: 200}
	entry.setCurrent(210)
	if entry.priceText != "" || entry.percentText != "" {
		t.Errorf("fields filled without input: price %q, percent %q", entry.priceText, entry.percentText)
	}
}
