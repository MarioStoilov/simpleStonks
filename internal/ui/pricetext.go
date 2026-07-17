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
	text *canvas.Text
	bg   *canvas.Rectangle
	anim *fyne.Animation

	last    float64
	hasLast bool
}

// newPriceText builds an empty ("—") price display.
func newPriceText() *priceText {
	p := &priceText{}
	p.text = canvas.NewText("—", theme.Color(theme.ColorNameForeground))
	p.text.TextStyle = fyne.TextStyle{Bold: true}
	p.text.Alignment = fyne.TextAlignTrailing
	p.bg = canvas.NewRectangle(color.Transparent)
	p.bg.CornerRadius = 3
	p.ExtendBaseWidget(p)
	return p
}

// SetPrice displays v. When flash is set, a change versus the previously
// shown price flashes the number's background (green rise, red drop); an
// unchanged price — e.g. polling a closed market — never flashes.
func (p *priceText) SetPrice(v float64, flash bool) {
	if do, rising := flashDirection(flash && p.hasLast, p.last, v); do {
		p.startFlash(rising)
	}
	p.last, p.hasLast = v, true
	p.text.Text = fmt.Sprintf("%.2f", v)
	p.Refresh()
}

// SetUnavailable shows the placeholder and forgets the last price.
func (p *priceText) SetUnavailable() {
	p.stopFlash()
	p.hasLast = false
	p.text.Text = "—"
	p.Refresh()
}

// flashDirection reports whether a price update should flash and whether it is
// a rise. ok gates on having a previous price to compare against.
func flashDirection(ok bool, last, v float64) (flash, rising bool) {
	if !ok || v == last {
		return false, false
	}
	return true, v > last
}

// startFlash begins (or restarts) the fade-out of the flash background.
func (p *priceText) startFlash(rising bool) {
	p.stopFlash()
	from := colorDown
	if rising {
		from = colorUp
	}
	from.A = flashAlpha
	to := from
	to.A = 0
	p.anim = canvas.NewColorRGBAAnimation(from, to, flashDuration, func(c color.Color) {
		p.bg.FillColor = c
		p.bg.Refresh()
	})
	p.anim.Start()
}

// stopFlash cancels any running flash and clears the background.
func (p *priceText) stopFlash() {
	if p.anim != nil {
		p.anim.Stop()
		p.anim = nil
	}
	p.bg.FillColor = color.Transparent
	p.bg.Refresh()
}

// CreateRenderer implements fyne.Widget.
func (p *priceText) CreateRenderer() fyne.WidgetRenderer {
	return &priceTextRenderer{p: p}
}

// priceTextRenderer spans the text across the widget and keeps the flash
// rectangle exactly behind the right-aligned number.
type priceTextRenderer struct{ p *priceText }

func (r *priceTextRenderer) Layout(size fyne.Size) {
	r.p.text.Resize(size)
	ts := fyne.MeasureText(r.p.text.Text, r.p.text.TextSize, r.p.text.TextStyle)
	r.p.bg.Resize(fyne.NewSize(ts.Width+flashPad, ts.Height))
	r.p.bg.Move(fyne.NewPos(size.Width-ts.Width-flashPad, 0))
}

func (r *priceTextRenderer) MinSize() fyne.Size { return r.p.text.MinSize() }

func (r *priceTextRenderer) Refresh() {
	r.Layout(r.p.Size())
	r.p.text.Refresh()
	r.p.bg.Refresh()
}

func (r *priceTextRenderer) Objects() []fyne.CanvasObject {
	return []fyne.CanvasObject{r.p.bg, r.p.text}
}

func (r *priceTextRenderer) Destroy() { r.p.stopFlash() }
