package qtui

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	qt "github.com/mappu/miqt/qt6"

	"github.com/MarioStoilov/simplestonks/internal/config"
	"github.com/MarioStoilov/simplestonks/internal/constants"
	"github.com/MarioStoilov/simplestonks/internal/notify"
)

// parseAlertPrice parses a price-alert entry: a plain positive decimal with
// at most constants.AlertPriceDecimals digits after the dot.
func parseAlertPrice(text string) (float64, bool) {
	value, ok := parseDecimal(strings.TrimSpace(text), constants.AlertPriceDecimals)
	if !ok || value <= 0 {
		return 0, false
	}
	return value, true
}

// parseAlertPercent parses a percentage-change entry: a decimal with an
// optional sign and "%" suffix and at most constants.AlertPercentDecimals
// digits after the dot. It must stay above -100%, where the price it stands
// for would no longer be positive.
func parseAlertPercent(text string) (float64, bool) {
	trimmed := strings.TrimSpace(strings.TrimSuffix(strings.TrimSpace(text), constants.SuffixPercent))
	negative := strings.HasPrefix(trimmed, "-")
	if negative || strings.HasPrefix(trimmed, "+") {
		trimmed = trimmed[1:]
	}
	value, ok := parseDecimal(trimmed, constants.AlertPercentDecimals)
	if !ok {
		return 0, false
	}
	if negative {
		value = -value
	}
	if value <= -constants.PercentMax {
		return 0, false
	}
	return value, true
}

// parseDecimal parses an unsigned plain decimal ("12", "12.5") with at most
// decimals digits after the dot.
func parseDecimal(text string, decimals int) (float64, bool) {
	whole, fraction, hasDot := strings.Cut(text, ".")
	if !isDigits(whole) || (hasDot && (!isDigits(fraction) || len(fraction) > decimals)) {
		return 0, false
	}
	value, err := strconv.ParseFloat(text, 64)
	if err != nil {
		return 0, false
	}
	return value, true
}

// isDigits reports whether text is one or more ASCII digits.
func isDigits(text string) bool {
	if text == "" {
		return false
	}
	for _, character := range text {
		if character < '0' || character > '9' {
			return false
		}
	}
	return true
}

// alertPillText renders a pending alert for its pill: the direction marker
// and threshold, followed — once a current price is known — by the
// threshold's distance from it in percent, e.g. "▲ 250.00 (+4.17%)".
func alertPillText(alert notify.Alert, current float64, known bool) string {
	marker := constants.SymbolAlertUp
	if !alert.Above {
		marker = constants.SymbolAlertDown
	}
	if !known || current <= 0 {
		return fmt.Sprintf(constants.FmtAlertPill, marker, alert.Price)
	}
	percent := (alert.Price - current) / current * constants.PercentMax
	return fmt.Sprintf(constants.FmtAlertPillDelta, marker, alert.Price, percent)
}

// alertAnchor is which of the alert dialog's two linked fields is active —
// the one the user last clicked into or typed in; the other one is derived
// from it and the current price.
type alertAnchor int

const (
	anchorNone alertAnchor = iota
	anchorPrice
	anchorPercent
)

// alertEntry is the state behind the alert dialog's linked price and
// percentage fields. The active field (the anchor) keeps its text and is
// never written to, so the cursor in it is left alone; the other is derived
// from it and follows the current price as it ticks: an entered price keeps
// its value while its percentage moves, an entered percentage keeps its
// value while its price moves.
type alertEntry struct {
	current     float64
	anchor      alertAnchor
	priceText   string
	percentText string
}

// focus makes a field the anchor as the user enters it, keeping both texts
// as they are: the value under the cursor freezes, and the other field
// starts following from the next edit or price tick.
func (entry *alertEntry) focus(anchor alertAnchor) {
	entry.anchor = anchor
}

// editPrice takes a user edit of the price field and derives the percentage.
func (entry *alertEntry) editPrice(text string) {
	entry.anchor = anchorPrice
	entry.priceText = text
	entry.derive()
}

// editPercent takes a user edit of the percentage field and derives the price.
func (entry *alertEntry) editPercent(text string) {
	entry.anchor = anchorPercent
	entry.percentText = text
	entry.derive()
}

// setCurrent takes a fresh current price and re-derives the field that
// follows it.
func (entry *alertEntry) setCurrent(current float64) {
	entry.current = current
	entry.derive()
}

// derive recomputes the non-anchor field: it is emptied while the anchor
// holds an invalid entry, and left alone while the anchor is empty (a field
// just entered must not wipe the value in the other one).
func (entry *alertEntry) derive() {
	switch entry.anchor {
	case anchorPrice:
		if strings.TrimSpace(entry.priceText) == "" {
			return
		}
		entry.percentText = ""
		if value, ok := parseAlertPrice(entry.priceText); ok && entry.current > 0 {
			percent := (value - entry.current) / entry.current * constants.PercentMax
			entry.percentText = fmt.Sprintf(constants.FmtAlertPercent, percent)
		}
	case anchorPercent:
		if strings.TrimSpace(entry.percentText) == "" {
			return
		}
		entry.priceText = ""
		if percent, ok := parseAlertPercent(entry.percentText); ok && entry.current > 0 {
			value := entry.current * (1 + percent/constants.PercentMax)
			entry.priceText = strconv.FormatFloat(value, 'f', constants.AlertPriceDecimals, 64)
		}
	}
}

// price is the alert price the fields stand for; ok is false while they do
// not hold a valid entry.
func (entry *alertEntry) price() (float64, bool) {
	return parseAlertPrice(entry.priceText)
}

// direction classifies the entered change for the theme's up/down colors
// (flat while there is no valid entry).
func (entry *alertEntry) direction() string {
	value, ok := entry.price()
	if !ok || entry.current <= 0 {
		return directionFlat
	}
	return directionOf(value - entry.current)
}

// showAlertDialog collects a new price alert for symbol. The alert can be
// entered as a price or as a percentage change from the current price; the
// two fields are linked (see alertEntry): the one clicked into or typed in
// is the active one, marked by the theme, and the percentage is colored
// like a price change. follow registers the dialog for the live price while it is
// open, so the current-price line and the derived field keep ticking. An
// optional expiry date, picked from a calendar popup, drops the alert once
// that day is over. Add validates
// positivity, precision, uniqueness, and the expiry before persisting through
// the store — the detail view's alert list re-renders via the config
// subscription.
func showAlertDialog(parent *qt.QWidget, store *config.Store, symbol string, current float64, follow func(listener func(price float64))) {
	dialog, body := newCardDialog(parent, fmt.Sprintf(constants.FmtTitleAlert, symbol))
	dialog.Resize(int(constants.AlertDialogWidth), int(constants.AlertDialogHeight))
	loaded := loadForm(alertForm)
	body.AddWidget(loaded.root)
	body.SetStretchFactor(loaded.root, 1)

	currentLabel := loaded.label("currentLabel")
	priceEdit := loaded.lineEdit("priceEdit")
	percentEdit := loaded.lineEdit("percentEdit")
	expiryEdit := loaded.lineEdit("expiryEdit")
	expiryClearButton := loaded.button("expiryClearButton")
	errorLabel := loaded.label("errorLabel")

	entry := &alertEntry{current: current}
	// show writes the state into the form. Only the derived field's text is
	// set, so the field being typed in keeps its cursor.
	show := func() {
		currentLabel.SetText(fmt.Sprintf(constants.FmtAlertCurrent, entry.current))
		switch entry.anchor {
		case anchorPrice:
			percentEdit.SetText(entry.percentText)
		case anchorPercent:
			priceEdit.SetText(entry.priceText)
		}
		setState(percentEdit.QWidget, "direction", entry.direction())
		setState(priceEdit.QWidget, "anchor", fmt.Sprint(entry.anchor == anchorPrice))
		setState(percentEdit.QWidget, "anchor", fmt.Sprint(entry.anchor == anchorPercent))
	}
	show()

	// Entering a field makes it the active one, so it is no longer written
	// to and its cursor stays where the user put it.
	priceEdit.OnFocusInEvent(func(super func(event *qt.QFocusEvent), event *qt.QFocusEvent) {
		super(event)
		entry.focus(anchorPrice)
		show()
	})
	percentEdit.OnFocusInEvent(func(super func(event *qt.QFocusEvent), event *qt.QFocusEvent) {
		super(event)
		entry.focus(anchorPercent)
		show()
	})

	// textEdited fires for user edits only, so writing the derived field
	// does not bounce back as an edit of it.
	priceEdit.OnTextEdited(func(text string) {
		errorLabel.SetText("")
		entry.editPrice(text)
		show()
	})
	percentEdit.OnTextEdited(func(text string) {
		errorLabel.SetText("")
		entry.editPercent(text)
		show()
	})
	// The expiry is empty until its field is clicked: the click sets today
	// and opens the calendar to pick another day; the clear button empties
	// it again.
	setExpiry := func(expires string) {
		errorLabel.SetText("")
		expiryEdit.SetText(expires)
		expiryClearButton.SetEnabled(expires != "")
	}
	setExpiry("")
	expiryEdit.OnMousePressEvent(func(super func(event *qt.QMouseEvent), event *qt.QMouseEvent) {
		today := dayOf(time.Now())
		if expiryEdit.Text() == "" {
			setExpiry(today.Format(constants.DateFmtISO))
		}
		selected, err := time.Parse(constants.DateFmtISO, expiryEdit.Text())
		if err != nil {
			selected = today
		}
		showCalendarPopup(expiryEdit.QWidget, selected, today, func(picked time.Time) {
			setExpiry(picked.Format(constants.DateFmtISO))
		})
	})
	expiryClearButton.OnClicked(func() { setExpiry("") })
	follow(func(price float64) {
		entry.setCurrent(price)
		show()
	})
	defer follow(nil)

	loaded.button("cancelButton").OnClicked(func() { dialog.Reject() })
	loaded.button("addButton").OnClicked(func() {
		value, ok := entry.price()
		if !ok {
			message := constants.MsgErrAlertPrice
			if entry.anchor == anchorPercent {
				message = constants.MsgErrAlertPct
			}
			errorLabel.SetText(message)
			return
		}
		if value == entry.current {
			errorLabel.SetText(constants.MsgErrAlertSame)
			return
		}
		for _, existing := range store.Get().Alerts {
			if existing.Symbol == symbol && existing.Price == value {
				errorLabel.SetText(constants.MsgErrAlertDup)
				return
			}
		}
		expires, ok := notify.ParseExpiry(expiryEdit.Text())
		if !ok {
			errorLabel.SetText(constants.MsgErrAlertExpiry)
			return
		}
		alert := notify.Alert{Symbol: symbol, Price: value, Above: value > entry.current, Expires: expires}
		if alert.ExpiredAt(time.Now()) {
			errorLabel.SetText(constants.MsgErrAlertExpiryPast)
			return
		}
		if err := store.Update(func(conf *config.Config) {
			conf.Alerts = append(conf.Alerts, alert)
		}); err != nil {
			errorLabel.SetText(err.Error())
			return
		}
		dialog.Accept()
	})

	dialog.Exec()
}
