package ui

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"

	"github.com/MarioStoilov/simplestonks/internal/constants"
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
	change := canvas.NewText("", constants.ColorNeutral)

	name := result.Symbol
	if result.Name != "" {
		name += constants.SepTitle + result.Name
	}
	title := widget.NewLabelWithStyle(name, fyne.TextAlignLeading, fyne.TextStyle{Bold: true})

	meta := result.Exchange
	if result.Type != "" {
		if meta != "" {
			meta += constants.SepMeta
		}
		meta += result.Type
	}
	subtitle := widget.NewLabel(meta)

	header := container.NewVBox(
		container.NewBorder(nil, nil, title, container.NewHBox(price, change)),
		subtitle,
	)

	var dlg *dialog.CustomDialog
	cancel := widget.NewButton(constants.LabelCancel, func() { dlg.Hide() })

	var addObj fyne.CanvasObject
	if containsStr(app.cfg.Symbols, result.Symbol) {
		addObj = disabledAddControl(constants.TipAlreadyTracked)
	} else {
		add := widget.NewButton(constants.LabelAdd, func() {
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
	dlg = dialog.NewCustomWithoutButtons(constants.TitlePreview, body, app.win)
	dlg.Resize(fyne.NewSize(constants.PreviewDialogWidth, constants.PreviewDialogHeight))
	dlg.Show()

	// Fetch the 1D series for the preview chart and header. The title already
	// carries the friendly name, so no name text is passed and nothing flashes.
	loadMainChart(app.provider, chartW, nil, price, change, result.Symbol, model.Range1D, false)
}

// disabledAddControl renders a grayed-out "Add" button that shows a tooltip on
// hover (used when the index is already tracked). Fyne's disabled buttons don't
// support tooltips, so this is a styled stand-in wrapped in a hoverTip.
func disabledAddControl(tip string) fyne.CanvasObject {
	boxBg := canvas.NewRectangle(constants.ColorDisabledBg)
	boxBg.CornerRadius = constants.PanelCornerRadius
	boxBg.SetMinSize(fyne.NewSize(constants.DisabledAddWidth, constants.DisabledAddHeight))
	label := canvas.NewText(constants.LabelAdd, constants.ColorDisabledFg)
	label.Alignment = fyne.TextAlignCenter
	content := container.NewStack(boxBg, container.NewCenter(label))
	return newHoverTip(content, tip)
}
