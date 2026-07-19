package ui

import (
	"context"
	"fmt"
	"image/color"
	"strings"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/widget"

	"github.com/MarioStoilov/simplestonks/internal/model"
	"github.com/MarioStoilov/simplestonks/internal/provider"
)

// searchDebounce is how long to wait after the last keystroke before searching.
const searchDebounce = 300 * time.Millisecond

// showSearchDialog opens the live-search add dialog: typing queries the provider
// (debounced) and shows suggestions annotated with full name and market; picking
// one adds it to the tracked list.
func (app *App) showSearchDialog() {
	if app.win == nil {
		return
	}
	entry := widget.NewEntry()
	entry.SetPlaceHolder("Search name or symbol — e.g. Apple or AAPL")
	status := widget.NewLabel("Start typing to search…")
	results := container.NewVBox()
	scroll := container.NewVScroll(results)
	scroll.SetMinSize(fyne.NewSize(440, 280))

	content := container.NewBorder(entry, status, nil, nil, scroll)
	dlg := dialog.NewCustom("Add index", "Close", content, app.win)
	dlg.Resize(fyne.NewSize(500, 440))

	controller := &searchController{
		provider: app.provider,
		results:  results,
		status:   status,
		// Selecting a result opens a preview; adding from there closes the
		// whole search dialog.
		onSelect: func(result model.SearchResult) {
			app.showResultPreview(result, func() { dlg.Hide() })
		},
	}
	entry.OnChanged = controller.onChanged

	dlg.Show()
	app.win.Canvas().Focus(entry)
}

// searchController manages the debounced live search inside the add dialog. All
// its methods run on the UI goroutine (OnChanged and the fyne.Do callback), so
// no locking is needed; a generation counter drops stale responses.
type searchController struct {
	provider provider.QuoteProvider
	results  *fyne.Container
	status   *widget.Label
	onSelect func(model.SearchResult)

	timer *time.Timer
	gen   int
}

func (controller *searchController) onChanged(query string) {
	query = strings.TrimSpace(query)
	controller.gen++
	gen := controller.gen
	if controller.timer != nil {
		controller.timer.Stop()
	}
	if query == "" {
		controller.status.SetText("Start typing to search…")
		controller.clear()
		return
	}
	controller.status.SetText("Searching…")
	controller.timer = time.AfterFunc(searchDebounce, func() {
		ctx, cancel := context.WithTimeout(context.Background(), fetchTimeout)
		defer cancel()
		res, err := controller.provider.Search(ctx, query)
		fyne.Do(func() {
			if gen != controller.gen {
				return // superseded by a newer keystroke
			}
			controller.render(res, err)
		})
	})
}

func (controller *searchController) clear() {
	controller.results.Objects = nil
	controller.results.Refresh()
}

func (controller *searchController) render(res []model.SearchResult, err error) {
	controller.clear()
	switch {
	case err != nil:
		controller.status.SetText("Search failed (offline?)")
		return
	case len(res) == 0:
		controller.status.SetText("No matches")
		return
	}
	controller.status.SetText(fmt.Sprintf("%d result(s)", len(res)))
	for _, result := range res {
		result := result
		controller.results.Add(newResultRow(result, func() { controller.onSelect(result) }))
	}
	controller.results.Refresh()
}

// resultRow renders one search suggestion (bold symbol + name over the
// market/type). It highlights on hover and opens the preview when tapped; there
// is no per-row add button.
type resultRow struct {
	widget.BaseWidget
	background *canvas.Rectangle
	root       fyne.CanvasObject
	onTap      func()
}

func newResultRow(result model.SearchResult, onTap func()) *resultRow {
	row := &resultRow{onTap: onTap}
	row.ExtendBaseWidget(row)

	titleText := result.Symbol
	if result.Name != "" {
		titleText += "  ·  " + result.Name
	}
	title := widget.NewLabelWithStyle(titleText, fyne.TextAlignLeading, fyne.TextStyle{Bold: true})

	sub := result.Exchange
	if result.Type != "" {
		if sub != "" {
			sub += " · "
		}
		sub += result.Type
	}
	subtitle := widget.NewLabel(sub)

	row.background = canvas.NewRectangle(color.Transparent)
	row.background.CornerRadius = 4
	row.root = container.NewStack(row.background, container.NewPadded(container.NewVBox(title, subtitle)))
	return row
}

func (row *resultRow) CreateRenderer() fyne.WidgetRenderer { return widget.NewSimpleRenderer(row.root) }

func (row *resultRow) Tapped(*fyne.PointEvent) {
	if row.onTap != nil {
		row.onTap()
	}
}

// Hoverable: generic highlight while the pointer is over the row.
func (row *resultRow) MouseIn(*desktop.MouseEvent) {
	row.background.FillColor = colorHover
	row.background.Refresh()
}

func (row *resultRow) MouseMoved(*desktop.MouseEvent) {}

func (row *resultRow) MouseOut() {
	row.background.FillColor = color.Transparent
	row.background.Refresh()
}
