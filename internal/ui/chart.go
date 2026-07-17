package ui

import (
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/widget"

	"github.com/MarioStoilov/simplestonks/internal/model"
)

const chartPadding = 4

// chart is a custom Fyne widget that plots a price series as a line, colored by
// the caller (up/down).
type chart struct {
	widget.BaseWidget
	series model.Series
	line   color.Color
}

// newChart constructs an empty chart widget.
func newChart() *chart {
	c := &chart{line: colorNeutral}
	c.ExtendBaseWidget(c)
	return c
}

// SetSeries updates the plotted data and redraws.
func (c *chart) SetSeries(s model.Series) {
	c.series = s
	c.Refresh()
}

// SetColor sets the line color used on the next redraw.
func (c *chart) SetColor(col color.Color) { c.line = col }

// CreateRenderer implements fyne.Widget.
func (c *chart) CreateRenderer() fyne.WidgetRenderer {
	bg := canvas.NewRectangle(color.NRGBA{R: 0x1e, G: 0x1e, B: 0x24, A: 0xff})
	return &chartRenderer{chart: c, bg: bg, objects: []fyne.CanvasObject{bg}}
}

type chartRenderer struct {
	chart   *chart
	bg      *canvas.Rectangle
	objects []fyne.CanvasObject
}

func (r *chartRenderer) Layout(size fyne.Size) {
	r.bg.Resize(size)
	r.rebuild(size)
}

func (r *chartRenderer) MinSize() fyne.Size { return fyne.NewSize(120, 80) }

func (r *chartRenderer) Refresh() {
	r.rebuild(r.chart.Size())
	canvas.Refresh(r.chart)
}

func (r *chartRenderer) Objects() []fyne.CanvasObject { return r.objects }

func (r *chartRenderer) Destroy() {}

// rebuild regenerates the line segments for the current series and size.
func (r *chartRenderer) rebuild(size fyne.Size) {
	objs := []fyne.CanvasObject{r.bg}
	pts := plotPath(closesOf(r.chart.series), size.Width, size.Height, chartPadding)
	for i := 1; i < len(pts); i++ {
		ln := canvas.NewLine(r.chart.line)
		ln.StrokeWidth = 1.5
		ln.Position1 = pts[i-1]
		ln.Position2 = pts[i]
		objs = append(objs, ln)
	}
	r.objects = objs
}

// closesOf extracts the closing prices from a series.
func closesOf(s model.Series) []float64 {
	out := make([]float64, len(s.Candles))
	for i, c := range s.Candles {
		out[i] = c.Close
	}
	return out
}

// plotPath maps close values to pixel positions within a w×h box (inset by pad),
// spacing points evenly on x and scaling y to the value range, inverted so higher
// values sit toward the top. It returns nil when there is nothing meaningful to
// draw (fewer than two points, or no room after padding).
func plotPath(values []float64, w, h, pad float32) []fyne.Position {
	if len(values) < 2 {
		return nil
	}
	min, max := values[0], values[0]
	for _, v := range values {
		if v < min {
			min = v
		}
		if v > max {
			max = v
		}
	}
	iw := float64(w) - 2*float64(pad)
	ih := float64(h) - 2*float64(pad)
	if iw <= 0 || ih <= 0 {
		return nil
	}

	valRange := max - min
	n := float64(len(values) - 1)
	pts := make([]fyne.Position, len(values))
	for i, v := range values {
		x := float64(pad) + iw*float64(i)/n
		yn := 0.5 // flat series sits in the middle
		if valRange != 0 {
			yn = (v - min) / valRange
		}
		y := float64(pad) + ih*(1-yn)
		pts[i] = fyne.NewPos(float32(x), float32(y))
	}
	return pts
}
