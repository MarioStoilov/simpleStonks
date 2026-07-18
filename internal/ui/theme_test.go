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
	for _, c := range cases {
		got, ok := parseHexColor(c.in)
		if ok != c.ok || got != c.want {
			t.Errorf("parseHexColor(%q) = (%v, %v), want (%v, %v)", c.in, got, ok, c.want, c.ok)
		}
	}
}

func TestFormatHexColorRoundTrip(t *testing.T) {
	in := color.NRGBA{R: 0x1a, G: 0xb2, B: 0x03, A: 0xff}
	s := formatHexColor(in)
	if s != "#1ab203" {
		t.Errorf("formatHexColor = %q, want %q", s, "#1ab203")
	}
	back, ok := parseHexColor(s)
	if !ok || back != in {
		t.Errorf("round trip = (%v, %v), want (%v, true)", back, ok, in)
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
	if a := backgroundColor(config.Background{Color: "#ffffff", Opacity: 2}).A; a != 0xff {
		t.Errorf("opacity > 1: alpha = %d, want 255", a)
	}
	if a := backgroundColor(config.Background{Color: "#ffffff", Opacity: -1}).A; a != 0 {
		t.Errorf("opacity < 0: alpha = %d, want 0", a)
	}
}
