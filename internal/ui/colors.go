package ui

import "image/color"

// Up/down/neutral colors used for price changes and chart lines.
var (
	colorUp      = color.NRGBA{R: 0x26, G: 0xa6, B: 0x5b, A: 0xff}
	colorDown    = color.NRGBA{R: 0xd0, G: 0x3a, B: 0x3a, A: 0xff}
	colorNeutral = color.NRGBA{R: 0x8a, G: 0x8a, B: 0x8a, A: 0xff}
)
