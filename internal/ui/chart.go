package ui

import (
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/widget"

	"github.com/MarioStoilov/simplestonks/internal/model"
)

// chart is a custom Fyne widget that plots a price series as a line. During
// scaffolding it renders a placeholder background; the plotting logic lands in
// v1 via SetSeries + the renderer's Refresh.
type chart struct {
	widget.BaseWidget
	series model.Series
}

// newChart constructs an empty chart widget.
func newChart() *chart {
	c := &chart{}
	c.ExtendBaseWidget(c)
	return c
}

// SetSeries updates the plotted data and triggers a redraw.
func (c *chart) SetSeries(s model.Series) {
	c.series = s
	c.Refresh()
}

// CreateRenderer implements fyne.Widget.
func (c *chart) CreateRenderer() fyne.WidgetRenderer {
	bg := canvas.NewRectangle(color.NRGBA{R: 0x1e, G: 0x1e, B: 0x24, A: 0xff})
	placeholder := canvas.NewText("chart", color.NRGBA{R: 0x88, G: 0x88, B: 0x88, A: 0xff})
	placeholder.Alignment = fyne.TextAlignCenter
	return &chartRenderer{
		chart:       c,
		bg:          bg,
		placeholder: placeholder,
		objects:     []fyne.CanvasObject{bg, placeholder},
	}
}

type chartRenderer struct {
	chart       *chart
	bg          *canvas.Rectangle
	placeholder *canvas.Text
	objects     []fyne.CanvasObject
}

func (r *chartRenderer) Layout(size fyne.Size) {
	r.bg.Resize(size)
	r.placeholder.Resize(fyne.NewSize(size.Width, r.placeholder.MinSize().Height))
	r.placeholder.Move(fyne.NewPos(0, (size.Height-r.placeholder.MinSize().Height)/2))
}

func (r *chartRenderer) MinSize() fyne.Size { return fyne.NewSize(120, 80) }

// Refresh rebuilds the drawing from the chart's series.
//
// TODO(v1): map r.chart.series.Candles into a canvas polyline scaled to the
// widget size, and color it by net change.
func (r *chartRenderer) Refresh() { canvas.Refresh(r.chart) }

func (r *chartRenderer) Objects() []fyne.CanvasObject { return r.objects }

func (r *chartRenderer) Destroy() {}
