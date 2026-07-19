package qtui

import (
	"fmt"
	"time"

	qt "github.com/mappu/miqt/qt6"

	"github.com/MarioStoilov/simplestonks/internal/constants"
	"github.com/MarioStoilov/simplestonks/internal/model"
)

// tile is one tracked-symbol card in the home grid: symbol + friendly name,
// the latest price with its change (colored), and a mini 1D chart. A live
// price change flashes the price background green/red.
type tile struct {
	frame  *qt.QFrame
	symbol string

	nameLabel   *qt.QLabel
	priceLabel  *qt.QLabel
	changeLabel *qt.QLabel
	changeBase  string // base stylesheet, differs for compact tiles
	chart       *chartWidget

	shownPrice float64
	priceShown bool
	flashTimer *qt.QTimer

	// Edit-mode reorder/remove controls (home grid only).
	controls        *qt.QWidget
	moveLeftButton  *qt.QPushButton
	moveRightButton *qt.QPushButton
	onMoveLeft      func()
	onMoveRight     func()
	onRemove        func()
}

const priceBaseStyle = "background: transparent; font-weight: 600;"

// tileStyle renders the tile stylesheet; a selected tile keeps its highlight
// even under the pointer (selected > hovered > card, as in the Fyne tile).
func tileStyle(selected bool) string {
	base := constants.ColorCardBg
	hover := constants.ColorHover
	if selected {
		base = constants.ColorSelected
		hover = constants.ColorSelected
	}
	return fmt.Sprintf(
		"#tile { background-color: %s; border-radius: %dpx; } #tile:hover { background-color: %s; }",
		cssRGB(base), int(constants.TileCornerRadius), cssRGB(hover))
}

// newTile builds an empty tile; onOpen fires when the tile is clicked, and
// compact shrinks it for the detail view's sidebar.
func newTile(parent *qt.QWidget, symbol string, compact bool, onOpen func()) *tile {
	frame := qt.NewQFrame(parent)
	frame.SetObjectName(*qt.NewQAnyStringView3("tile"))
	frame.SetStyleSheet(tileStyle(false))
	if compact {
		// Width follows the sidebar viewport; only the height is fixed.
		frame.SetMinimumHeight(int(constants.SidebarTileHeight))
	} else {
		frame.SetMinimumSize2(int(constants.GridCellWidth), int(constants.GridCellHeight))
	}

	layout := qt.NewQVBoxLayout(frame.QWidget)

	head := qt.NewQHBoxLayout2()
	ident := qt.NewQVBoxLayout2()
	symbolLabel := qt.NewQLabel5(symbol, frame.QWidget)
	symbolLabel.SetStyleSheet("background: transparent; font-weight: 600;")
	nameLabel := qt.NewQLabel(frame.QWidget)
	nameLabel.SetStyleSheet(fmt.Sprintf("background: transparent; color: %s; font-size: %dpx;",
		cssRGB(constants.ColorAxis), int(constants.NameTextSize)))
	// A long friendly name must never widen the tile beyond its cell: let the
	// label be clipped instead of driving the layout's minimum width.
	nameLabel.SetSizePolicy2(qt.QSizePolicy__Ignored, qt.QSizePolicy__Preferred)
	ident.AddWidget(symbolLabel.QWidget)
	ident.AddWidget(nameLabel.QWidget)
	head.AddLayout(ident.QLayout)
	head.AddStretch()

	quote := qt.NewQVBoxLayout2()
	priceLabel := qt.NewQLabel5(constants.PricePlaceholder, frame.QWidget)
	priceLabel.SetStyleSheet(priceBaseStyle)
	priceLabel.SetAlignment(qt.AlignRight)
	changeLabel := qt.NewQLabel(frame.QWidget)
	changeStyleSheet := "background: transparent;"
	if compact {
		// The sidebar is narrow: a smaller change text keeps price + change
		// inside the tile.
		changeStyleSheet = fmt.Sprintf("background: transparent; font-size: %dpx;", int(constants.NameTextSize))
	}
	changeLabel.SetStyleSheet(changeStyleSheet)
	changeLabel.SetAlignment(qt.AlignRight)
	quote.AddWidget(priceLabel.QWidget)
	quote.AddWidget(changeLabel.QWidget)
	head.AddLayout(quote.QLayout)
	layout.AddLayout(head.QLayout)

	chart := newChartWidget(frame.QWidget)
	layout.AddWidget2(chart.QWidget, 1)

	// Edit-mode controls: reorder left/right and remove; hidden until the
	// home grid enters edit mode.
	controls := qt.NewQWidget(frame.QWidget)
	controls.SetStyleSheet("background: transparent;")
	controlsLayout := qt.NewQHBoxLayout(controls)
	controlsLayout.SetContentsMargins(0, 0, 0, 0)
	cell := &tile{}
	moveLeftButton := qt.NewQPushButton5("◀", controls)
	moveLeftButton.SetStyleSheet(dialogButtonStyle(false))
	moveLeftButton.SetCursor(qt.NewQCursor2(qt.PointingHandCursor))
	moveLeftButton.OnClicked(func() {
		if cell.onMoveLeft != nil {
			cell.onMoveLeft()
		}
	})
	controlsLayout.AddWidget(moveLeftButton.QWidget)
	moveRightButton := qt.NewQPushButton5("▶", controls)
	moveRightButton.SetStyleSheet(dialogButtonStyle(false))
	moveRightButton.SetCursor(qt.NewQCursor2(qt.PointingHandCursor))
	moveRightButton.OnClicked(func() {
		if cell.onMoveRight != nil {
			cell.onMoveRight()
		}
	})
	controlsLayout.AddWidget(moveRightButton.QWidget)
	controlsLayout.AddStretch()
	removeButton := qt.NewQPushButton5("✕", controls)
	removeButton.SetStyleSheet(windowButtonStyle(cssRGB(constants.ColorDown)))
	removeButton.SetCursor(qt.NewQCursor2(qt.PointingHandCursor))
	removeButton.OnClicked(func() {
		if cell.onRemove != nil {
			cell.onRemove()
		}
	})
	controlsLayout.AddWidget(removeButton.QWidget)
	controls.Hide()
	layout.AddWidget(controls)

	flashTimer := qt.NewQTimer2(frame.QObject)
	flashTimer.SetSingleShot(true)
	flashTimer.OnTimeout(func() {
		priceLabel.SetStyleSheet(priceBaseStyle)
	})

	if onOpen != nil {
		frame.OnMousePressEvent(func(super func(event *qt.QMouseEvent), event *qt.QMouseEvent) {
			onOpen()
		})
	}

	cell.frame = frame
	cell.symbol = symbol
	cell.nameLabel = nameLabel
	cell.priceLabel = priceLabel
	cell.changeLabel = changeLabel
	cell.changeBase = changeStyleSheet
	cell.chart = chart
	cell.flashTimer = flashTimer
	cell.controls = controls
	cell.moveLeftButton = moveLeftButton
	cell.moveRightButton = moveRightButton
	return cell
}

// setEditing shows or hides the reorder/remove controls.
func (cell *tile) setEditing(editing bool) {
	cell.controls.SetVisible(editing)
}

// setMoveBounds enables the reorder buttons that have somewhere to go.
func (cell *tile) setMoveBounds(canLeft, canRight bool) {
	cell.moveLeftButton.SetEnabled(canLeft)
	cell.moveRightButton.SetEnabled(canRight)
}

// setSeries updates the tile from a fetched series. With flash enabled, a
// changed price (after the first display) flashes the price background in the
// movement's direction.
func (cell *tile) setSeries(series model.Series, flash bool) {
	cell.nameLabel.SetText(series.Name)

	last := series.Candles[len(series.Candles)-1].Close
	cell.priceLabel.SetText(chartPrice(last))

	lineColor := constants.ColorNeutral
	if series.PreviousClose > 0 {
		change := last - series.PreviousClose
		percent := change / series.PreviousClose * constants.PercentMax
		col, sign := changeStyle(change)
		lineColor = col
		cell.changeLabel.SetText(fmt.Sprintf(constants.FmtPriceChange, sign, change, sign, percent))
		cell.changeLabel.SetStyleSheet(cell.changeBase + " color: " + cssRGB(col) + ";")
	} else {
		cell.changeLabel.SetText("")
	}
	cell.chart.setSeries(series, lineColor)

	if flash && cell.priceShown && last != cell.shownPrice {
		flashColor := constants.ColorUp
		if last < cell.shownPrice {
			flashColor = constants.ColorDown
		}
		cell.priceLabel.SetStyleSheet(fmt.Sprintf("%s background-color: %s; border-radius: %dpx;",
			priceBaseStyle, cssRGBA(flashColor, constants.FlashAlpha), int(constants.FlashCornerRadius)))
		cell.flashTimer.Start(int(constants.FlashDuration / time.Millisecond))
	}
	cell.shownPrice = last
	cell.priceShown = true
}

// setSelected toggles the sidebar highlight.
func (cell *tile) setSelected(selected bool) {
	cell.frame.SetStyleSheet(tileStyle(selected))
}

// setFailed marks the tile as having no data.
func (cell *tile) setFailed() {
	cell.priceLabel.SetText(constants.PricePlaceholder)
	cell.changeLabel.SetText(constants.MsgUnavailable)
	cell.changeLabel.SetStyleSheet(cell.changeBase + " color: " + cssRGB(constants.ColorNeutral) + ";")
	cell.chart.setSeries(model.Series{}, constants.ColorNeutral)
	cell.priceShown = false
}

// chartPrice formats a price for the tile header.
func chartPrice(value float64) string {
	return fmt.Sprintf(constants.FmtPrice, value)
}
