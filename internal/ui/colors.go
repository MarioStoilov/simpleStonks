package ui

import (
	"image/color"

	"github.com/MarioStoilov/simplestonks/internal/constants"
)

// changeStyle returns the color and numeric-sign prefix for a price delta.
func changeStyle(delta float64) (col color.Color, sign string) {
	switch {
	case delta > 0:
		return constants.ColorUp, "+"
	case delta < 0:
		return constants.ColorDown, "" // the number already carries a minus sign
	default:
		return constants.ColorNeutral, ""
	}
}
