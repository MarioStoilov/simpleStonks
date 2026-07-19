package ui

import (
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"

	"github.com/MarioStoilov/simplestonks/internal/model"
)

// showResultPreview opens a small preview of a search result — a compact version
// of the detail view (symbol/name header, price/change, a 1D chart) — with Add
// and Cancel actions. If the index is already tracked, Add is disabled and shows
// a tooltip on hover. Adding tracks the symbol and calls onAdded (used to also
// close the search dialog).
func (app *App) showResultPreview(result model.SearchResult, onAdded func()) {
	if app.win == nil {
		return
	}

	chartW := newChart()
	price := newPriceText()
	change := canvas.NewText("", colorNeutral)

	name := result.Symbol
	if result.Name != "" {
		name += "  ·  " + result.Name
	}
	title := widget.NewLabelWithStyle(name, fyne.TextAlignLeading, fyne.TextStyle{Bold: true})

	meta := result.Exchange
	if result.Type != "" {
		if meta != "" {
			meta += " · "
		}
		meta += result.Type
	}
	subtitle := widget.NewLabel(meta)

	header := container.NewVBox(
		container.NewBorder(nil, nil, title, container.NewHBox(price, change)),
		subtitle,
	)

	var dlg *dialog.CustomDialog
	cancel := widget.NewButton("Cancel", func() { dlg.Hide() })

	var addObj fyne.CanvasObject
	if containsStr(app.cfg.Symbols, result.Symbol) {
		addObj = disabledAddControl("Already tracking this index")
	} else {
		add := widget.NewButton("Add", func() {
			app.addSymbol(result.Symbol)
			dlg.Hide()
			if onAdded != nil {
				onAdded()
			}
		})
		add.Importance = widget.HighImportance
		addObj = add
	}
	footer := container.NewHBox(layout.NewSpacer(), cancel, addObj)

	body := container.NewBorder(header, footer, nil, nil, chartW)
	dlg = dialog.NewCustomWithoutButtons("Preview", body, app.win)
	dlg.Resize(fyne.NewSize(520, 440))
	dlg.Show()

	// Fetch the 1D series for the preview chart and header. The title already
	// carries the friendly name, so no name text is passed and nothing flashes.
	loadMainChart(app.provider, chartW, nil, price, change, result.Symbol, model.Range1D, false)
}

// disabledAddControl renders a grayed-out "Add" button that shows a tooltip on
// hover (used when the index is already tracked). Fyne's disabled buttons don't
// support tooltips, so this is a styled stand-in wrapped in a hoverTip.
func disabledAddControl(tip string) fyne.CanvasObject {
	boxBg := canvas.NewRectangle(color.NRGBA{R: 0x3a, G: 0x3a, B: 0x40, A: 0xff})
	boxBg.CornerRadius = 4
	boxBg.SetMinSize(fyne.NewSize(72, 32))
	label := canvas.NewText("Add", color.NRGBA{R: 0x80, G: 0x80, B: 0x80, A: 0xff})
	label.Alignment = fyne.TextAlignCenter
	content := container.NewStack(boxBg, container.NewCenter(label))
	return newHoverTip(content, tip)
}
