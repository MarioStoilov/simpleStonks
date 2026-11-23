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
