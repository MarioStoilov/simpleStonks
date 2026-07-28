package qtui

import "testing"

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
