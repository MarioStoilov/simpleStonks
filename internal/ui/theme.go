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
	base       fyne.Theme
	background color.NRGBA
}

func (custom appTheme) Color(name fyne.ThemeColorName, variant fyne.ThemeVariant) color.Color {
	if name == theme.ColorNameBackground {
		return custom.background
	}
	return custom.base.Color(name, variant)
}

func (custom appTheme) Font(style fyne.TextStyle) fyne.Resource    { return custom.base.Font(style) }
func (custom appTheme) Icon(name fyne.ThemeIconName) fyne.Resource { return custom.base.Icon(name) }
func (custom appTheme) Size(name fyne.ThemeSizeName) float32       { return custom.base.Size(name) }

// backgroundColor resolves a config.Background to a concrete color, falling
// back to the default color for unparsable hex strings and clamping opacity.
func backgroundColor(background config.Background) color.NRGBA {
	resolved, ok := parseHexColor(background.Color)
	if !ok {
		resolved, _ = parseHexColor(config.DefaultBackground().Color)
	}
	opacity := background.Opacity
	if opacity < 0 {
		opacity = 0
	}
	if opacity > 1 {
		opacity = 1
	}
	resolved.A = uint8(opacity*255 + 0.5)
	return resolved
}

// formatHexColor renders a color in the "#RRGGBB" hex form used in config.
func formatHexColor(col color.NRGBA) string {
	return fmt.Sprintf("#%02x%02x%02x", col.R, col.G, col.B)
}

// parseHexColor parses a "#RRGGBB" hex color (the "#" is optional), returning
// it fully opaque.
func parseHexColor(hex string) (color.NRGBA, bool) {
	hex = strings.TrimPrefix(strings.TrimSpace(hex), "#")
	if len(hex) != 6 {
		return color.NRGBA{}, false
	}
	var channels [3]uint8
	for idx := 0; idx < 3; idx++ {
		high, ok1 := hexNibble(hex[2*idx])
		low, ok2 := hexNibble(hex[2*idx+1])
		if !ok1 || !ok2 {
			return color.NRGBA{}, false
		}
		channels[idx] = high<<4 | low
	}
	return color.NRGBA{R: channels[0], G: channels[1], B: channels[2], A: 0xff}, true
}

func hexNibble(char byte) (uint8, bool) {
	switch {
	case char >= '0' && char <= '9':
		return char - '0', true
	case char >= 'a' && char <= 'f':
		return char - 'a' + 10, true
	case char >= 'A' && char <= 'F':
		return char - 'A' + 10, true
	}
	return 0, false
}

// applyBackground applies the saved background config to the running app,
// skipping the (whole-UI) theme refresh when nothing changed.
func (app *App) applyBackground() {
	if app.bgApplied == app.cfg.Background && app.bgSet {
		return
	}
	app.bgApplied, app.bgSet = app.cfg.Background, true
	app.setBackground(backgroundColor(app.cfg.Background))
}

// previewBackground applies an unsaved background live (settings preview).
func (app *App) previewBackground(background config.Background) {
	app.bgApplied, app.bgSet = background, true
	app.setBackground(backgroundColor(background))
}

func (app *App) setBackground(col color.NRGBA) {
	app.fyne.Settings().SetTheme(appTheme{base: theme.DefaultTheme(), background: col})
}
