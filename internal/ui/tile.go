package ui

import (
	"fmt"
	"image/color"
	"log/slog"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/MarioStoilov/simplestonks/internal/model"
)

// nameTextSize is the font size of the small friendly-name line shown under a
// symbol (on tiles and the detail header).
const nameTextSize = 11

// tile is a tappable card for one symbol: the symbol with its friendly name,
// its latest price and percent change, and — when showChart is set — a mini 1D
// chart. It is used both for home-grid cells (with chart) and detail-view
// sidebar cells (without).
type tile struct {
	widget.BaseWidget
	symbol string
	onTap  func()

	selected bool
	hovered  bool

	bg     *canvas.Rectangle
	name   *canvas.Text // friendly name, filled in once a series arrives
	price  *priceText
	change *canvas.Text
	chart  *chart // nil when showChart is false
	root   fyne.CanvasObject
}

// newTile builds a tile. onTap fires when the cell (outside its buttons) is
// clicked; onRemove, when non-nil, adds a small remove button.
func newTile(symbol string, showChart bool, onTap, onRemove func()) *tile {
	t := &tile{symbol: symbol, onTap: onTap}
	t.ExtendBaseWidget(t)

	symLbl := widget.NewLabelWithStyle(symbol, fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
	t.name = canvas.NewText("", colorAxis)
	t.name.TextSize = nameTextSize
	t.price = newPriceText()
	t.change = canvas.NewText("", colorNeutral)

	var right fyne.CanvasObject = t.price
	if onRemove != nil {
		rm := widget.NewButtonWithIcon("", theme.ContentClearIcon(), onRemove)
		rm.Importance = widget.LowImportance
		right = container.NewHBox(t.price, rm)
	}
	header := container.NewBorder(nil, nil, symLbl, right)

	var content fyne.CanvasObject
	if showChart {
		t.chart = newChart()
		// The chart is the topmost hoverable over most of the cell, so it
		// forwards its hover state to keep the tile's effect seamless.
		t.chart.onHover = t.setHovered
		content = container.NewBorder(container.NewVBox(header, t.name, t.change), nil, nil, nil, t.chart)
	} else {
		content = container.NewVBox(header, t.name, t.change)
	}

	t.bg = canvas.NewRectangle(colorCardBg)
	t.bg.CornerRadius = 6
	t.root = container.NewStack(t.bg, container.NewPadded(content))
	return t
}

// CreateRenderer implements fyne.Widget.
func (t *tile) CreateRenderer() fyne.WidgetRenderer { return widget.NewSimpleRenderer(t.root) }

// Tapped implements fyne.Tappable.
func (t *tile) Tapped(*fyne.PointEvent) {
	if t.onTap != nil {
		t.onTap()
	}
}

// MouseIn implements desktop.Hoverable.
func (t *tile) MouseIn(*desktop.MouseEvent) { t.setHovered(true) }

// MouseMoved implements desktop.Hoverable.
func (t *tile) MouseMoved(*desktop.MouseEvent) {}

// MouseOut implements desktop.Hoverable.
func (t *tile) MouseOut() { t.setHovered(false) }

// setHovered applies the button-like hover highlight, signalling that the
// tile is clickable — so it is skipped for non-tappable (edit mode) tiles.
func (t *tile) setHovered(h bool) {
	if t.onTap == nil || t.hovered == h {
		return
	}
	t.hovered = h
	t.updateBg()
}

// SetSelected highlights the tile (used for the current sidebar entry).
func (t *tile) SetSelected(sel bool) {
	t.selected = sel
	t.updateBg()
}

// updateBg resolves the background from the selection/hover state.
func (t *tile) updateBg() {
	switch {
	case t.selected:
		t.bg.FillColor = colorSelected
	case t.hovered:
		t.bg.FillColor = colorHover
	default:
		t.bg.FillColor = colorCardBg
	}
	t.bg.Refresh()
}

// setSeries renders a fetched series onto the tile.
func (t *tile) setSeries(s model.Series) {
	last, prev, ok := latestAndPrev(s)
	if !ok {
		t.setError(fmt.Errorf("no data"))
		return
	}
	col, text := priceChangeText(last, prev)
	if s.Name != "" && s.Name != t.name.Text {
		t.name.Text = s.Name
		t.name.Refresh()
	}
	t.price.SetPrice(last, true) // flashes only when a shown price changes
	t.change.Text = text
	t.change.Color = col
	t.change.Refresh()
	if t.chart != nil {
		t.chart.SetColor(col)
		t.chart.SetSeries(s)
	}
}

// setError puts the tile into an unavailable state and logs the cause.
func (t *tile) setError(err error) {
	t.price.SetUnavailable()
	t.change.Text = "unavailable"
	t.change.Color = colorNeutral
	t.change.Refresh()
	if t.chart != nil {
		t.chart.SetColor(colorNeutral)
		t.chart.SetSeries(model.Series{})
	}
	slog.Warn("tile update failed", "symbol", t.symbol, "err", err)
}

// latestAndPrev returns the last close and the reference previous close.
func latestAndPrev(s model.Series) (last, prev float64, ok bool) {
	if len(s.Candles) == 0 {
		return 0, 0, false
	}
	return s.Candles[len(s.Candles)-1].Close, s.PreviousClose, true
}

// priceChangeText formats the "+1.23 (+0.45%)" change string and its color.
func priceChangeText(last, prev float64) (color.Color, string) {
	delta := last - prev
	col, sign := changeStyle(delta)
	pct := 0.0
	if prev != 0 {
		pct = delta / prev * 100
	}
	return col, fmt.Sprintf("%s%.2f (%s%.2f%%)", sign, delta, sign, pct)
}
