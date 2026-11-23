package qtui

import (
	"image/color"
	"testing"

	"github.com/MarioStoilov/simplestonks/internal/colorscheme"
)

func TestResolveColorScheme(t *testing.T) {
	cases := []struct {
		preference  colorscheme.Scheme
		paletteDark bool
		want        string
	}{
		{colorscheme.Unknown, false, colorSchemeLight}, // the default
		{colorscheme.Unknown, true, colorSchemeDark},   // no preference: the palette decides
		{colorscheme.Dark, false, colorSchemeDark},     // a stated preference wins
		{colorscheme.Light, true, colorSchemeLight},
	}
	for _, testCase := range cases {
		if got := resolveColorScheme(testCase.preference, testCase.paletteDark); got != testCase.want {
			t.Errorf("resolveColorScheme(%v, %v) = %q, want %q",
				testCase.preference, testCase.paletteDark, got, testCase.want)
		}
	}
}

func TestPaletteIsDark(t *testing.T) {
	white := color.NRGBA{R: 0xff, G: 0xff, B: 0xff, A: 0xff}
	black := color.NRGBA{A: 0xff}
	charcoal := color.NRGBA{R: 0x2a, G: 0x2e, B: 0x32, A: 0xff}
	if paletteIsDark(white, black) {
		t.Error("black text on a white window read as dark")
	}
	if !paletteIsDark(charcoal, white) {
		t.Error("white text on a charcoal window read as light")
	}
}
