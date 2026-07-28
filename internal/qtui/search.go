package qtui

import (
	"context"
	"fmt"
	"strings"

	qt "github.com/mappu/miqt/qt6"
	"github.com/mappu/miqt/qt6/mainthread"

	"github.com/MarioStoilov/simplestonks/internal/config"
	"github.com/MarioStoilov/simplestonks/internal/constants"
	"github.com/MarioStoilov/simplestonks/internal/model"
	"github.com/MarioStoilov/simplestonks/internal/provider"
)

// showSearchDialog runs the add-symbol live search: typing queries the
// provider (debounced, with a generation counter dropping stale responses)
// and lists suggestions; picking one opens the preview, and adding from there
// closes the whole dialog.
func showSearchDialog(parent *qt.QWidget, quotes provider.QuoteProvider, store *config.Store) {
	dialog, body := newCardDialog(parent, constants.TitleAddIndex)
	dialog.Resize(int(constants.SearchDialogWidth), int(constants.SearchDialogHeight))

	query := qt.NewQLineEdit(dialog.QWidget)
	query.SetPlaceholderText(constants.PlaceholderSearch)
	query.SetStyleSheet(inputStyle())
	body.AddWidget(query.QWidget)

	results := qt.NewQListWidget(dialog.QWidget)
	results.SetMinimumSize2(int(constants.SearchScrollMinWidth), int(constants.SearchScrollMinHeight))
	results.SetStyleSheet(fmt.Sprintf(constants.StyleSearchResults,
		int(constants.PanelCornerRadius), cssRGB(constants.ColorHover)))
	body.AddWidget2(results.QWidget, 1)

	status := qt.NewQLabel5(constants.MsgSearchPrompt, dialog.QWidget)
	status.SetStyleSheet("background: transparent; color: " + cssRGB(constants.ColorNeutral) + ";")
	body.AddWidget(status.QWidget)

	closed := false
	dialog.OnFinished(func(result int) { closed = true })

	var found []model.SearchResult
	generation := 0

	debounce := qt.NewQTimer2(dialog.QObject)
	debounce.SetSingleShot(true)
	debounce.OnTimeout(func() {
		trimmed := strings.TrimSpace(query.Text())
		if trimmed == "" {
			return
		}
		generation++
		requestGen := generation
		go func() {
			ctx, cancel := context.WithTimeout(context.Background(), constants.FetchTimeout)
			defer cancel()
			matches, err := quotes.Search(ctx, trimmed)
			mainthread.Wait(func() {
				if closed || requestGen != generation {
					return // dialog gone or superseded by a newer keystroke
				}
				results.Clear()
				if err != nil {
					found = nil
					status.SetText(constants.MsgSearchFailed)
					return
				}
				found = matches
				if len(matches) == 0 {
					status.SetText(constants.MsgNoMatches)
					return
				}
				status.SetText(fmt.Sprintf(constants.FmtResultCount, len(matches)))
				for _, match := range matches {
					item := qt.NewQListWidgetItem()
					row := newResultRow(results.QWidget, match)
					item.SetSizeHint(row.SizeHint())
					results.AddItemWithItem(item)
					results.SetItemWidget(item, row)
				}
			})
		}()
	})

	query.OnTextChanged(func(text string) {
		generation++ // invalidate any in-flight search immediately
		if strings.TrimSpace(text) == "" {
			results.Clear()
			found = nil
			status.SetText(constants.MsgSearchPrompt)
			return
		}
		status.SetText(constants.MsgSearching)
		debounce.Start(int(constants.SearchDebounce.Milliseconds()))
	})

	results.OnItemClicked(func(item *qt.QListWidgetItem) {
		row := results.Row(item)
		if row < 0 || row >= len(found) {
			return
		}
		if showPreviewDialog(dialog.QWidget, quotes, store, found[row]) {
			dialog.Accept()
		}
	})

	query.SetFocus()
	dialog.Exec()
}

// newResultRow builds a two-line suggestion row: bold "SYMBOL · Name" over a
// gray "exchange · type" line.
func newResultRow(parent *qt.QWidget, match model.SearchResult) *qt.QWidget {
	row := qt.NewQWidget(parent)
	row.SetStyleSheet("background: transparent;")
	layout := qt.NewQVBoxLayout(row)

	title := match.Symbol
	if match.Name != "" {
		title += constants.SepTitle + match.Name
	}
	titleLabel := qt.NewQLabel5(title, row)
	titleLabel.SetStyleSheet("background: transparent; font-weight: 600;")
	titleLabel.SetSizePolicy2(qt.QSizePolicy__Ignored, qt.QSizePolicy__Preferred)
	layout.AddWidget(titleLabel.QWidget)

	var meta []string
	if match.Exchange != "" {
		meta = append(meta, match.Exchange)
	}
	if match.Type != "" {
		meta = append(meta, match.Type)
	}
	if len(meta) > 0 {
		metaLabel := qt.NewQLabel5(strings.Join(meta, constants.SepMeta), row)
		metaLabel.SetStyleSheet(fmt.Sprintf(constants.StyleSmallText,
			cssRGB(constants.ColorNeutral), int(constants.NameTextSize)))
		metaLabel.SetSizePolicy2(qt.QSizePolicy__Ignored, qt.QSizePolicy__Preferred)
		layout.AddWidget(metaLabel.QWidget)
	}
	return row
}
