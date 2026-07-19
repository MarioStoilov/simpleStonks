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

	background *canvas.Rectangle
	name       *canvas.Text // friendly name, filled in once a series arrives
	price      *priceText
	change     *canvas.Text
	chart      *chart // nil when showChart is false
	root       fyne.CanvasObject
}

// newTile builds a tile. onTap fires when the cell (outside its buttons) is
// clicked; onRemove, when non-nil, adds a small remove button.
func newTile(symbol string, showChart bool, onTap, onRemove func()) *tile {
	cell := &tile{symbol: symbol, onTap: onTap}
	cell.ExtendBaseWidget(cell)

	symLbl := widget.NewLabelWithStyle(symbol, fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
	cell.name = canvas.NewText("", colorAxis)
	cell.name.TextSize = nameTextSize
	cell.price = newPriceText()
	cell.change = canvas.NewText("", colorNeutral)

	var right fyne.CanvasObject = cell.price
	if onRemove != nil {
		removeBtn := widget.NewButtonWithIcon("", theme.ContentClearIcon(), onRemove)
		removeBtn.Importance = widget.LowImportance
		right = container.NewHBox(cell.price, removeBtn)
	}
	header := container.NewBorder(nil, nil, symLbl, right)

	var content fyne.CanvasObject
	if showChart {
		cell.chart = newChart()
		// The chart is the topmost hoverable over most of the cell, so it
		// forwards its hover state to keep the tile's effect seamless.
		cell.chart.onHover = cell.setHovered
		content = container.NewBorder(container.NewVBox(header, cell.name, cell.change), nil, nil, nil, cell.chart)
	} else {
		content = container.NewVBox(header, cell.name, cell.change)
	}

	cell.background = canvas.NewRectangle(colorCardBg)
	cell.background.CornerRadius = 6
	cell.root = container.NewStack(cell.background, container.NewPadded(content))
	return cell
}

// CreateRenderer implements fyne.Widget.
func (cell *tile) CreateRenderer() fyne.WidgetRenderer { return widget.NewSimpleRenderer(cell.root) }

// Tapped implements fyne.Tappable.
func (cell *tile) Tapped(*fyne.PointEvent) {
	if cell.onTap != nil {
		cell.onTap()
	}
}

// MouseIn implements desktop.Hoverable.
func (cell *tile) MouseIn(*desktop.MouseEvent) { cell.setHovered(true) }

// MouseMoved implements desktop.Hoverable.
func (cell *tile) MouseMoved(*desktop.MouseEvent) {}

// MouseOut implements desktop.Hoverable.
func (cell *tile) MouseOut() { cell.setHovered(false) }

// setHovered applies the button-like hover highlight, signalling that the
// tile is clickable — so it is skipped for non-tappable (edit mode) tiles.
func (cell *tile) setHovered(hovered bool) {
	if cell.onTap == nil || cell.hovered == hovered {
		return
	}
	cell.hovered = hovered
	cell.updateBg()
}

// SetSelected highlights the tile (used for the current sidebar entry).
func (cell *tile) SetSelected(selected bool) {
	cell.selected = selected
	cell.updateBg()
}

// updateBg resolves the background from the selection/hover state.
func (cell *tile) updateBg() {
	switch {
	case cell.selected:
		cell.background.FillColor = colorSelected
	case cell.hovered:
		cell.background.FillColor = colorHover
	default:
		cell.background.FillColor = colorCardBg
	}
	cell.background.Refresh()
}

// setSeries renders a fetched series onto the tile.
func (cell *tile) setSeries(series model.Series) {
	last, prev, ok := latestAndPrev(series)
	if !ok {
		cell.setError(fmt.Errorf("no data"))
		return
	}
	col, text := priceChangeText(last, prev)
	if series.Name != "" && series.Name != cell.name.Text {
		cell.name.Text = series.Name
		cell.name.Refresh()
	}
	cell.price.SetPrice(last, true) // flashes only when a shown price changes
	cell.change.Text = text
	cell.change.Color = col
	cell.change.Refresh()
	if cell.chart != nil {
		cell.chart.SetColor(col)
		cell.chart.SetSeries(series)
	}
}

// setError puts the tile into an unavailable state and logs the cause.
func (cell *tile) setError(err error) {
	cell.price.SetUnavailable()
	cell.change.Text = "unavailable"
	cell.change.Color = colorNeutral
	cell.change.Refresh()
	if cell.chart != nil {
		cell.chart.SetColor(colorNeutral)
		cell.chart.SetSeries(model.Series{})
	}
	slog.Warn("tile update failed", "symbol", cell.symbol, "err", err)
}

// latestAndPrev returns the last close and the reference previous close.
func latestAndPrev(series model.Series) (last, prev float64, ok bool) {
	if len(series.Candles) == 0 {
		return 0, 0, false
	}
	return series.Candles[len(series.Candles)-1].Close, series.PreviousClose, true
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
