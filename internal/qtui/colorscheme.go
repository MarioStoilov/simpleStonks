package qtui

import (
	"image/color"

	qt "github.com/mappu/miqt/qt6"

	"github.com/MarioStoilov/simplestonks/internal/colorscheme"
)

// The colorScheme property marks the window and dialog cards with the
// desktop's color scheme; the theme's tooltip rules key off it.
const (
	colorSchemeProperty = "colorScheme"
	colorSchemeLight    = "light"
	colorSchemeDark     = "dark"
)

// activeColorScheme is the scheme the cards currently carry (UI thread
// only): the window card follows it live, and each dialog card takes it when
// the dialog opens.
var activeColorScheme = colorSchemeLight

// resolveColorScheme picks the cards' scheme: the desktop's stated
// preference when it has one, otherwise whether the desktop palette itself
// is dark. Light is the default.
func resolveColorScheme(preference colorscheme.Scheme, paletteDark bool) string {
	switch {
	case preference == colorscheme.Dark:
		return colorSchemeDark
	case preference == colorscheme.Light:
		return colorSchemeLight
	case paletteDark:
		return colorSchemeDark
	default:
		return colorSchemeLight
	}
}

// paletteIsDark reports whether a palette reads as a dark theme: its window
// text is brighter than the window background behind it.
func paletteIsDark(window, text color.NRGBA) bool {
	return brightness(text) > brightness(window)
}

// brightness is a color's summed channel intensity — enough to order a
// palette's text against its background.
func brightness(src color.NRGBA) int {
	return int(src.R) + int(src.G) + int(src.B)
}

// desktopPaletteIsDark applies paletteIsDark to the application palette,
// which Qt takes from the desktop's platform theme.
func desktopPaletteIsDark() bool {
	palette := qt.QGuiApplication_Palette()
	return paletteIsDark(
		nrgba(palette.ColorWithCr(qt.QPalette__Window)),
		nrgba(palette.ColorWithCr(qt.QPalette__WindowText)))
}

// nrgba converts an opaque QColor to a palette color.
func nrgba(src *qt.QColor) color.NRGBA {
	return color.NRGBA{R: uint8(src.Red()), G: uint8(src.Green()), B: uint8(src.Blue()), A: 0xff}
}
