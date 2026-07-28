package qtui

import (
	"fmt"
	"image/color"
	"strconv"
	"strings"

	qt "github.com/mappu/miqt/qt6"

	"github.com/MarioStoilov/simplestonks/internal/constants"
)

// qColor converts a palette color to a QColor.
func qColor(src color.NRGBA) *qt.QColor {
	out := qt.NewQColor3(int(src.R), int(src.G), int(src.B))
	out.SetAlpha(int(src.A))
	return out
}

// cssRGB renders a palette color as a CSS rgb() literal for stylesheets.
func cssRGB(src color.NRGBA) string {
	return fmt.Sprintf("rgb(%d, %d, %d)", src.R, src.G, src.B)
}

// cssRGBA renders a palette color with an explicit alpha (0..255) as a CSS
// rgba() literal for stylesheets.
func cssRGBA(src color.NRGBA, alpha uint8) string {
	return fmt.Sprintf("rgba(%d, %d, %d, %d)", src.R, src.G, src.B, alpha)
}

// alphaByte converts a 0..1 opacity into an alpha channel byte.
func alphaByte(opacity float64) uint8 {
	return uint8(opacity*float64(0xff) + 0.5)
}

// scrollAreaStyle is the stylesheet for transparent scroll areas with slim
// dark scrollbars matching the widget look (replaces the chunky theme ones).
func scrollAreaStyle() string {
	return fmt.Sprintf(constants.StyleScrollArea,
		int(constants.ScrollBarWidth), int(constants.ScrollBarWidth),
		cssRGB(constants.ColorHover), int(constants.PanelCornerRadius),
		int(constants.ScrollBarMinHandle), cssRGB(constants.ColorSelected))
}

// changeStyle returns the color and sign prefix for a price change: green
// with "+" for gains, red for losses, neutral for no change.
func changeStyle(change float64) (color.NRGBA, string) {
	switch {
	case change > 0:
		return constants.ColorUp, "+"
	case change < 0:
		return constants.ColorDown, ""
	default:
		return constants.ColorNeutral, ""
	}
}

// parseHexColor parses "#RRGGBB" (the "#" and surrounding space optional,
// case-insensitive) into an opaque color; reports ok=false on malformed input.
func parseHexColor(hex string) (color.NRGBA, bool) {
	trimmed := strings.TrimPrefix(strings.TrimSpace(hex), "#")
	if len(trimmed) != len("RRGGBB") {
		return color.NRGBA{}, false
	}
	packed, err := strconv.ParseUint(trimmed, 16, 32)
	if err != nil {
		return color.NRGBA{}, false
	}
	return color.NRGBA{R: uint8(packed >> 16), G: uint8(packed >> 8), B: uint8(packed), A: 0xff}, true
}
