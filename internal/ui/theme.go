package ui

import (
	"fmt"
	"image/color"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/theme"

	"github.com/MarioStoilov/simplestonks/internal/config"
)

// appTheme wraps the default theme, overriding the window background with the
// configured color and opacity. Fyne premultiplies the alpha when clearing the
// canvas, so a lower opacity visibly fades the background (and becomes true
// see-through transparency on surfaces that support it).
type appTheme struct {
	base fyne.Theme
	bg   color.NRGBA
}

func (t appTheme) Color(n fyne.ThemeColorName, v fyne.ThemeVariant) color.Color {
	if n == theme.ColorNameBackground {
		return t.bg
	}
	return t.base.Color(n, v)
}

func (t appTheme) Font(s fyne.TextStyle) fyne.Resource     { return t.base.Font(s) }
func (t appTheme) Icon(n fyne.ThemeIconName) fyne.Resource { return t.base.Icon(n) }
func (t appTheme) Size(n fyne.ThemeSizeName) float32       { return t.base.Size(n) }

// backgroundColor resolves a config.Background to a concrete color, falling
// back to the default color for unparsable hex strings and clamping opacity.
func backgroundColor(b config.Background) color.NRGBA {
	c, ok := parseHexColor(b.Color)
	if !ok {
		c, _ = parseHexColor(config.DefaultBackground().Color)
	}
	op := b.Opacity
	if op < 0 {
		op = 0
	}
	if op > 1 {
		op = 1
	}
	c.A = uint8(op*255 + 0.5)
	return c
}

// formatHexColor renders a color in the "#RRGGBB" hex form used in config.
func formatHexColor(c color.NRGBA) string {
	return fmt.Sprintf("#%02x%02x%02x", c.R, c.G, c.B)
}

// parseHexColor parses a "#RRGGBB" hex color (the "#" is optional), returning
// it fully opaque.
func parseHexColor(s string) (color.NRGBA, bool) {
	s = strings.TrimPrefix(strings.TrimSpace(s), "#")
	if len(s) != 6 {
		return color.NRGBA{}, false
	}
	var v [3]uint8
	for i := 0; i < 3; i++ {
		hi, ok1 := hexNibble(s[2*i])
		lo, ok2 := hexNibble(s[2*i+1])
		if !ok1 || !ok2 {
			return color.NRGBA{}, false
		}
		v[i] = hi<<4 | lo
	}
	return color.NRGBA{R: v[0], G: v[1], B: v[2], A: 0xff}, true
}

func hexNibble(c byte) (uint8, bool) {
	switch {
	case c >= '0' && c <= '9':
		return c - '0', true
	case c >= 'a' && c <= 'f':
		return c - 'a' + 10, true
	case c >= 'A' && c <= 'F':
		return c - 'A' + 10, true
	}
	return 0, false
}

// applyBackground applies the saved background config to the running app,
// skipping the (whole-UI) theme refresh when nothing changed.
func (a *App) applyBackground() {
	if a.bgApplied == a.cfg.Background && a.bgSet {
		return
	}
	a.bgApplied, a.bgSet = a.cfg.Background, true
	a.setBackground(backgroundColor(a.cfg.Background))
}

// previewBackground applies an unsaved background live (settings preview).
func (a *App) previewBackground(b config.Background) {
	a.bgApplied, a.bgSet = b, true
	a.setBackground(backgroundColor(b))
}

func (a *App) setBackground(c color.NRGBA) {
	a.fyne.Settings().SetTheme(appTheme{base: theme.DefaultTheme(), bg: c})
}
