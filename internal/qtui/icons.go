package qtui

import (
	qt "github.com/mappu/miqt/qt6"
)

// svgPixmap renders embedded SVG bytes at a square size through Qt's SVG
// image plugin (the same path the About logo uses); a failed decode yields
// an empty pixmap, which Qt draws as nothing.
func svgPixmap(data []byte, size int) *qt.QPixmap {
	pixmap := qt.NewQPixmap()
	if !pixmap.LoadFromDataWithData(data) {
		return pixmap
	}
	return pixmap.Scaled3(size, size, qt.KeepAspectRatio, qt.SmoothTransformation)
}
