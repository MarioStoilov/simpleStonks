package ui

import "image/color"

// Palette. The app currently assumes a dark surface (the chart background is
// dark too); this is placeholder styling while the UI design is finalized.
var (
	colorUp       = color.NRGBA{R: 0x26, G: 0xa6, B: 0x5b, A: 0xff}
	colorDown     = color.NRGBA{R: 0xd0, G: 0x3a, B: 0x3a, A: 0xff}
	colorNeutral  = color.NRGBA{R: 0x8a, G: 0x8a, B: 0x8a, A: 0xff}
	colorCardBg   = color.NRGBA{R: 0x24, G: 0x26, B: 0x2e, A: 0xff}
	colorSelected = color.NRGBA{R: 0x30, G: 0x3a, B: 0x52, A: 0xff}
	colorHover    = color.NRGBA{R: 0x2c, G: 0x30, B: 0x3c, A: 0xff}
)

// changeStyle returns the color and numeric-sign prefix for a price delta.
func changeStyle(delta float64) (col color.Color, sign string) {
	switch {
	case delta > 0:
		return colorUp, "+"
	case delta < 0:
		return colorDown, "" // the number already carries a minus sign
	default:
		return colorNeutral, ""
	}
}
