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
func (a *App) showSearchDialog() {
	if a.win == nil {
		return
	}
	entry := widget.NewEntry()
	entry.SetPlaceHolder("Search name or symbol — e.g. Apple or AAPL")
	status := widget.NewLabel("Start typing to search…")
	results := container.NewVBox()
	scroll := container.NewVScroll(results)
	scroll.SetMinSize(fyne.NewSize(440, 280))

	content := container.NewBorder(entry, status, nil, nil, scroll)
	d := dialog.NewCustom("Add index", "Close", content, a.win)
	d.Resize(fyne.NewSize(500, 440))

	sc := &searchController{
		provider: a.provider,
		results:  results,
		status:   status,
		// Selecting a result opens a preview; adding from there closes the
		// whole search dialog.
		onSelect: func(r model.SearchResult) {
			a.showResultPreview(r, func() { d.Hide() })
		},
	}
	entry.OnChanged = sc.onChanged

	d.Show()
	a.win.Canvas().Focus(entry)
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

func (sc *searchController) onChanged(q string) {
	q = strings.TrimSpace(q)
	sc.gen++
	gen := sc.gen
	if sc.timer != nil {
		sc.timer.Stop()
	}
	if q == "" {
		sc.status.SetText("Start typing to search…")
		sc.clear()
		return
	}
	sc.status.SetText("Searching…")
	sc.timer = time.AfterFunc(searchDebounce, func() {
		ctx, cancel := context.WithTimeout(context.Background(), fetchTimeout)
		defer cancel()
		res, err := sc.provider.Search(ctx, q)
		fyne.Do(func() {
			if gen != sc.gen {
				return // superseded by a newer keystroke
			}
			sc.render(res, err)
		})
	})
}

func (sc *searchController) clear() {
	sc.results.Objects = nil
	sc.results.Refresh()
}

func (sc *searchController) render(res []model.SearchResult, err error) {
	sc.clear()
	switch {
	case err != nil:
		sc.status.SetText("Search failed (offline?)")
		return
	case len(res) == 0:
		sc.status.SetText("No matches")
		return
	}
	sc.status.SetText(fmt.Sprintf("%d result(s)", len(res)))
	for _, r := range res {
		r := r
		sc.results.Add(newResultRow(r, func() { sc.onSelect(r) }))
	}
	sc.results.Refresh()
}

// resultRow renders one search suggestion (bold symbol + name over the
// market/type). It highlights on hover and opens the preview when tapped; there
// is no per-row add button.
type resultRow struct {
	widget.BaseWidget
	bg    *canvas.Rectangle
	root  fyne.CanvasObject
	onTap func()
}

func newResultRow(r model.SearchResult, onTap func()) *resultRow {
	row := &resultRow{onTap: onTap}
	row.ExtendBaseWidget(row)

	titleText := r.Symbol
	if r.Name != "" {
		titleText += "  ·  " + r.Name
	}
	title := widget.NewLabelWithStyle(titleText, fyne.TextAlignLeading, fyne.TextStyle{Bold: true})

	sub := r.Exchange
	if r.Type != "" {
		if sub != "" {
			sub += " · "
		}
		sub += r.Type
	}
	subtitle := widget.NewLabel(sub)

	row.bg = canvas.NewRectangle(color.Transparent)
	row.bg.CornerRadius = 4
	row.root = container.NewStack(row.bg, container.NewPadded(container.NewVBox(title, subtitle)))
	return row
}

func (r *resultRow) CreateRenderer() fyne.WidgetRenderer { return widget.NewSimpleRenderer(r.root) }

func (r *resultRow) Tapped(*fyne.PointEvent) {
	if r.onTap != nil {
		r.onTap()
	}
}

// Hoverable: generic highlight while the pointer is over the row.
func (r *resultRow) MouseIn(*desktop.MouseEvent) {
	r.bg.FillColor = colorHover
	r.bg.Refresh()
}

func (r *resultRow) MouseMoved(*desktop.MouseEvent) {}

func (r *resultRow) MouseOut() {
	r.bg.FillColor = color.Transparent
	r.bg.Refresh()
}
