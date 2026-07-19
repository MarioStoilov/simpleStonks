package qtui

import (
	"image/color"
	"testing"
)

func TestParseHexColor(t *testing.T) {
	cases := []struct {
		in   string
		want color.NRGBA
		ok   bool
	}{
		{"#1a2B3c", color.NRGBA{R: 0x1a, G: 0x2b, B: 0x3c, A: 0xff}, true},
		{"1A2b3C", color.NRGBA{R: 0x1a, G: 0x2b, B: 0x3c, A: 0xff}, true},
		{"  #ffffff ", color.NRGBA{R: 0xff, G: 0xff, B: 0xff, A: 0xff}, true},
		{"#fff", color.NRGBA{}, false},    // short form not supported
		{"#12345g", color.NRGBA{}, false}, // bad digit
		{"", color.NRGBA{}, false},
		{"#1234567", color.NRGBA{}, false}, // too long
	}
	for _, testCase := range cases {
		got, ok := parseHexColor(testCase.in)
		if ok != testCase.ok || got != testCase.want {
			t.Errorf("parseHexColor(%q) = (%v, %v), want (%v, %v)", testCase.in, got, ok, testCase.want, testCase.ok)
		}
	}
}

func TestHexOfRoundTrip(t *testing.T) {
	original := color.NRGBA{R: 0x1a, G: 0xb2, B: 0x03, A: 0xff}
	hex := hexOf(original)
	if hex != "#1ab203" {
		t.Errorf("hexOf = %q, want %q", hex, "#1ab203")
	}
	back, ok := parseHexColor(hex)
	if !ok || back != original {
		t.Errorf("round trip = (%v, %v), want (%v, true)", back, ok, original)
	}
}

func TestParseWhole(t *testing.T) {
	if value, err := parseWhole(" 15 "); err != nil || value != 15 {
		t.Errorf("parseWhole(15) = (%d, %v), want (15, nil)", value, err)
	}
	if value, err := parseWhole("0"); err != nil || value != 0 {
		t.Errorf("parseWhole(0) = (%d, %v), want (0, nil)", value, err)
	}
	for _, invalid := range []string{"", "abc", "-1", "1.5"} {
		if _, err := parseWhole(invalid); err == nil {
			t.Errorf("parseWhole(%q) succeeded, want error", invalid)
		}
	}
}

func TestAlphaByte(t *testing.T) {
	if alpha := alphaByte(0); alpha != 0 {
		t.Errorf("alphaByte(0) = %d, want 0", alpha)
	}
	if alpha := alphaByte(1); alpha != 0xff {
		t.Errorf("alphaByte(1) = %d, want 255", alpha)
	}
	if alpha := alphaByte(0.5); alpha != 0x80 {
		t.Errorf("alphaByte(0.5) = %d, want 128", alpha)
	}
}
