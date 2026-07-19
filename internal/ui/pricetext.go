package ui

import (
	"fmt"
	"image/color"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

// flashDuration is how long a price-update flash takes to fade out.
const flashDuration = 900 * time.Millisecond

// flashAlpha is the starting opacity of the flash background.
const flashAlpha = 0x66

// flashPad is the extra background width to the left of the number.
const flashPad = 3

// priceText displays a right-aligned price number and, on live updates,
// flashes the number's background — semi-transparent green for a rise, red
// for a drop, fading out. The flash covers only the number itself.
type priceText struct {
	widget.BaseWidget
	text       *canvas.Text
	background *canvas.Rectangle
	anim       *fyne.Animation

	last    float64
	hasLast bool
}

// newPriceText builds an empty ("—") price display.
func newPriceText() *priceText {
	price := &priceText{}
	price.text = canvas.NewText("—", theme.Color(theme.ColorNameForeground))
	price.text.TextStyle = fyne.TextStyle{Bold: true}
	price.text.Alignment = fyne.TextAlignTrailing
	price.background = canvas.NewRectangle(color.Transparent)
	price.background.CornerRadius = 3
	price.ExtendBaseWidget(price)
	return price
}

// SetPrice displays value. When flash is set, a change versus the previously
// shown price flashes the number's background (green rise, red drop); an
// unchanged price — e.g. polling a closed market — never flashes.
func (price *priceText) SetPrice(value float64, flash bool) {
	if shouldFlash, rising := flashDirection(flash && price.hasLast, price.last, value); shouldFlash {
		price.startFlash(rising)
	}
	price.last, price.hasLast = value, true
	price.text.Text = fmt.Sprintf("%.2f", value)
	price.Refresh()
}

// SetUnavailable shows the placeholder and forgets the last price.
func (price *priceText) SetUnavailable() {
	price.stopFlash()
	price.hasLast = false
	price.text.Text = "—"
	price.Refresh()
}

// flashDirection reports whether a price update should flash and whether it is
// a rise. hasPrevious gates on having a previous price to compare against.
func flashDirection(hasPrevious bool, last, value float64) (flash, rising bool) {
	if !hasPrevious || value == last {
		return false, false
	}
	return true, value > last
}

// startFlash begins (or restarts) the fade-out of the flash background.
func (price *priceText) startFlash(rising bool) {
	price.stopFlash()
	start := colorDown
	if rising {
		start = colorUp
	}
	start.A = flashAlpha
	end := start
	end.A = 0
	price.anim = canvas.NewColorRGBAAnimation(start, end, flashDuration, func(faded color.Color) {
		price.background.FillColor = faded
		price.background.Refresh()
	})
	price.anim.Start()
}

// stopFlash cancels any running flash and clears the background.
func (price *priceText) stopFlash() {
	if price.anim != nil {
		price.anim.Stop()
		price.anim = nil
	}
	price.background.FillColor = color.Transparent
	price.background.Refresh()
}

// CreateRenderer implements fyne.Widget.
func (price *priceText) CreateRenderer() fyne.WidgetRenderer {
	return &priceTextRenderer{price: price}
}

// priceTextRenderer spans the text across the widget and keeps the flash
// rectangle exactly behind the right-aligned number.
type priceTextRenderer struct{ price *priceText }

func (renderer *priceTextRenderer) Layout(size fyne.Size) {
	renderer.price.text.Resize(size)
	textSize := fyne.MeasureText(renderer.price.text.Text, renderer.price.text.TextSize, renderer.price.text.TextStyle)
	renderer.price.background.Resize(fyne.NewSize(textSize.Width+flashPad, textSize.Height))
	renderer.price.background.Move(fyne.NewPos(size.Width-textSize.Width-flashPad, 0))
}

func (renderer *priceTextRenderer) MinSize() fyne.Size { return renderer.price.text.MinSize() }

func (renderer *priceTextRenderer) Refresh() {
	renderer.Layout(renderer.price.Size())
	renderer.price.text.Refresh()
	renderer.price.background.Refresh()
}

func (renderer *priceTextRenderer) Objects() []fyne.CanvasObject {
	return []fyne.CanvasObject{renderer.price.background, renderer.price.text}
}

func (renderer *priceTextRenderer) Destroy() { renderer.price.stopFlash() }
