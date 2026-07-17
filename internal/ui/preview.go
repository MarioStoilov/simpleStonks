package ui

import (
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/MarioStoilov/simplestonks/internal/model"
)

// showResultPreview opens a small preview of a search result — a compact version
// of the detail view (symbol/name header, price/change, a 1D chart) — with Add
// and Cancel actions. If the index is already tracked, Add is disabled and shows
// a tooltip on hover. Adding tracks the symbol and calls onAdded (used to also
// close the search dialog).
func (a *App) showResultPreview(r model.SearchResult, onAdded func()) {
	if a.win == nil {
		return
	}

	chartW := newChart()
	price := canvas.NewText("—", theme.Color(theme.ColorNameForeground))
	price.TextStyle = fyne.TextStyle{Bold: true}
	price.Alignment = fyne.TextAlignTrailing
	change := canvas.NewText("", colorNeutral)

	name := r.Symbol
	if r.Name != "" {
		name += "  ·  " + r.Name
	}
	title := widget.NewLabelWithStyle(name, fyne.TextAlignLeading, fyne.TextStyle{Bold: true})

	meta := r.Exchange
	if r.Type != "" {
		if meta != "" {
			meta += " · "
		}
		meta += r.Type
	}
	subtitle := widget.NewLabel(meta)

	header := container.NewVBox(
		container.NewBorder(nil, nil, title, container.NewHBox(price, change)),
		subtitle,
	)

	var d *dialog.CustomDialog
	cancel := widget.NewButton("Cancel", func() { d.Hide() })

	var addObj fyne.CanvasObject
	if containsStr(a.cfg.Symbols, r.Symbol) {
		addObj = disabledAddControl("Already tracking this index")
	} else {
		add := widget.NewButton("Add", func() {
			a.addSymbol(r.Symbol)
			d.Hide()
			if onAdded != nil {
				onAdded()
			}
		})
		add.Importance = widget.HighImportance
		addObj = add
	}
	footer := container.NewHBox(layout.NewSpacer(), cancel, addObj)

	body := container.NewBorder(header, footer, nil, nil, chartW)
	d = dialog.NewCustomWithoutButtons("Preview", body, a.win)
	d.Resize(fyne.NewSize(520, 440))
	d.Show()

	// Fetch the 1D series for the preview chart and header.
	loadMainChart(a.provider, chartW, price, change, r.Symbol, model.Range1D)
}

// disabledAddControl renders a grayed-out "Add" button that shows a tooltip on
// hover (used when the index is already tracked). Fyne's disabled buttons don't
// support tooltips, so this is a styled stand-in wrapped in a hoverTip.
func disabledAddControl(tip string) fyne.CanvasObject {
	bg := canvas.NewRectangle(color.NRGBA{R: 0x3a, G: 0x3a, B: 0x40, A: 0xff})
	bg.CornerRadius = 4
	bg.SetMinSize(fyne.NewSize(72, 32))
	label := canvas.NewText("Add", color.NRGBA{R: 0x80, G: 0x80, B: 0x80, A: 0xff})
	label.Alignment = fyne.TextAlignCenter
	content := container.NewStack(bg, container.NewCenter(label))
	return newHoverTip(content, tip)
}
