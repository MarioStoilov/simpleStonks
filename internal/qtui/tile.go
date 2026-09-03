package qtui

import (
	"fmt"
	"time"

	qt "github.com/mappu/miqt/qt6"

	"github.com/MarioStoilov/simplestonks/internal/constants"
	"github.com/MarioStoilov/simplestonks/internal/model"
)

// tile is one tracked-symbol card: symbol + friendly name, the latest price
// with its change, and a mini 1D chart. Its look comes from the theme; a
// live price change flashes the price background in the movement's
// direction. The grid tile (tile.ui) also carries the edit-mode controls;
// the sidebar tile (sidebar_tile.ui) is the compact variant.
type tile struct {
	frame  *qt.QFrame
	symbol string

	nameLabel   *qt.QLabel
	priceLabel  *qt.QLabel
	changeLabel *qt.QLabel
	chart       *chartWidget

	shownPrice float64
	priceShown bool
	flashTimer *qt.QTimer

	// Edit-mode reorder/remove controls (grid tile only).
	controls        *qt.QWidget
	moveLeftButton  *qt.QPushButton
	moveRightButton *qt.QPushButton
	onMoveLeft      func()
	onMoveRight     func()
	onRemove        func()
}

// newTile loads a tile form; onOpen fires when the tile is clicked, and
// compact picks the sidebar variant.
func newTile(parent *qt.QWidget, symbol string, compact bool, onOpen func()) *tile {
	formName := tileForm
	if compact {
		formName = sidebarTileForm
	}
	loaded := loadForm(formName)
	loaded.root.SetParent(parent)

	cell := &tile{
		frame:       loaded.frame("tile"),
		symbol:      symbol,
		nameLabel:   loaded.label("nameLabel"),
		priceLabel:  loaded.label("priceLabel"),
		changeLabel: loaded.label("changeLabel"),
		chart:       loaded.chart("chart"),
	}
	loaded.label("symbolLabel").SetText(symbol)
	setState(cell.frame.QWidget, "compact", fmt.Sprint(compact))

	if !compact {
		cell.controls = loaded.widget("controls")
		cell.controls.Hide() // until the home grid enters edit mode
		cell.moveLeftButton = loaded.button("moveLeftButton")
		cell.moveLeftButton.OnClicked(func() {
			if cell.onMoveLeft != nil {
				cell.onMoveLeft()
			}
		})
		cell.moveRightButton = loaded.button("moveRightButton")
		cell.moveRightButton.OnClicked(func() {
			if cell.onMoveRight != nil {
				cell.onMoveRight()
			}
		})
		loaded.button("removeButton").OnClicked(func() {
			if cell.onRemove != nil {
				cell.onRemove()
			}
		})
	}

	cell.flashTimer = qt.NewQTimer2(cell.frame.QObject)
	cell.flashTimer.SetSingleShot(true)
	cell.flashTimer.OnTimeout(func() {
		setState(cell.priceLabel.QWidget, "flash", "")
	})

	if onOpen != nil {
		cell.frame.OnMousePressEvent(func(super func(event *qt.QMouseEvent), event *qt.QMouseEvent) {
			onOpen()
		})
	}
	return cell
}

// setEditing shows or hides the reorder/remove controls.
func (cell *tile) setEditing(editing bool) {
	if cell.controls != nil {
		cell.controls.SetVisible(editing)
	}
}

// setMoveBounds enables the reorder buttons that have somewhere to go.
func (cell *tile) setMoveBounds(canLeft, canRight bool) {
	if cell.moveLeftButton != nil {
		cell.moveLeftButton.SetEnabled(canLeft)
		cell.moveRightButton.SetEnabled(canRight)
	}
}

// setSeries updates the tile from a fetched series. With flash enabled, a
// changed price (after the first display) flashes the price background in
// the movement's direction.
func (cell *tile) setSeries(series model.Series, flash bool) {
	cell.nameLabel.SetText(series.Name)

	last := series.Candles[len(series.Candles)-1].Close
	cell.priceLabel.SetText(chartPrice(last))

	lineColor := constants.ColorNeutral
	direction := directionFlat
	if series.PreviousClose > 0 {
		change := last - series.PreviousClose
		percent := change / series.PreviousClose * constants.PercentMax
		col, sign := changeStyle(change)
		lineColor = col
		direction = directionOf(change)
		cell.changeLabel.SetText(fmt.Sprintf(constants.FmtPriceChange, sign, change, sign, percent))
	} else {
		cell.changeLabel.SetText("")
	}
	setState(cell.changeLabel.QWidget, "direction", direction)
	cell.chart.setSeries(series, lineColor)

	if flash && cell.priceShown && last != cell.shownPrice {
		setState(cell.priceLabel.QWidget, "flash", directionOf(last-cell.shownPrice))
		cell.flashTimer.Start(int(constants.FlashDuration / time.Millisecond))
	}
	cell.shownPrice = last
	cell.priceShown = true
}

// setSelected toggles the sidebar highlight.
func (cell *tile) setSelected(selected bool) {
	setState(cell.frame.QWidget, "selected", fmt.Sprint(selected))
}

// setFailed marks the tile as having no data.
func (cell *tile) setFailed() {
	cell.priceLabel.SetText(constants.PricePlaceholder)
	cell.changeLabel.SetText(constants.MsgUnavailable)
	setState(cell.changeLabel.QWidget, "direction", directionFlat)
	cell.chart.setSeries(model.Series{}, constants.ColorNeutral)
	cell.priceShown = false
}

// Movement directions, as the theme's selectors expect them.
const (
	directionUp   = "up"
	directionDown = "down"
	directionFlat = "flat"
)

// directionOf classifies a price change for the theme.
func directionOf(change float64) string {
	switch {
	case change > 0:
		return directionUp
	case change < 0:
		return directionDown
	default:
		return directionFlat
	}
}

// chartPrice formats a price for the tile header.
func chartPrice(value float64) string {
	return fmt.Sprintf(constants.FmtPrice, value)
}
