package ui

import (
	"image/color"
	"testing"

	"github.com/MarioStoilov/simplestonks/internal/config"
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

func TestFormatHexColorRoundTrip(t *testing.T) {
	original := color.NRGBA{R: 0x1a, G: 0xb2, B: 0x03, A: 0xff}
	hex := formatHexColor(original)
	if hex != "#1ab203" {
		t.Errorf("formatHexColor = %q, want %q", hex, "#1ab203")
	}
	back, ok := parseHexColor(hex)
	if !ok || back != original {
		t.Errorf("round trip = (%v, %v), want (%v, true)", back, ok, original)
	}
}

func TestBackgroundColor(t *testing.T) {
	// Valid color with half opacity.
	got := backgroundColor(config.Background{Color: "#204060", Opacity: 0.5})
	want := color.NRGBA{R: 0x20, G: 0x40, B: 0x60, A: 128}
	if got != want {
		t.Errorf("got %v, want %v", got, want)
	}

	// Unparsable color falls back to the default color, keeping the opacity.
	def, _ := parseHexColor(config.DefaultBackground().Color)
	got = backgroundColor(config.Background{Color: "nope", Opacity: 1})
	def.A = 0xff
	if got != def {
		t.Errorf("fallback = %v, want default %v", got, def)
	}

	// Opacity is clamped to [0, 1].
	if alpha := backgroundColor(config.Background{Color: "#ffffff", Opacity: 2}).A; alpha != 0xff {
		t.Errorf("opacity > 1: alpha = %d, want 255", alpha)
	}
	if alpha := backgroundColor(config.Background{Color: "#ffffff", Opacity: -1}).A; alpha != 0 {
		t.Errorf("opacity < 0: alpha = %d, want 0", alpha)
	}
}
