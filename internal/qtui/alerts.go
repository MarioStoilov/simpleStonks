package qtui

import (
	"fmt"
	"strconv"
	"strings"

	qt "github.com/mappu/miqt/qt6"

	"github.com/MarioStoilov/simplestonks/internal/config"
	"github.com/MarioStoilov/simplestonks/internal/constants"
	"github.com/MarioStoilov/simplestonks/internal/notify"
)

// parseAlertPrice parses a price-alert entry: a plain positive decimal with
// at most constants.AlertPriceDecimals digits after the dot.
func parseAlertPrice(text string) (float64, bool) {
	trimmed := strings.TrimSpace(text)
	whole, fraction, hasDot := strings.Cut(trimmed, ".")
	if !isDigits(whole) || (hasDot && !isDigits(fraction)) {
		return 0, false
	}
	if hasDot && len(fraction) > constants.AlertPriceDecimals {
		return 0, false
	}
	value, err := strconv.ParseFloat(trimmed, 64)
	if err != nil || value <= 0 {
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

// showAlertDialog collects a new price alert for symbol against its current
// headline price: the entry live-previews its % distance from the current
// price (colored up/down), and Add validates positivity, precision, and
// uniqueness before persisting through the store — the detail view's alert
// list re-renders via the config subscription.
func showAlertDialog(parent *qt.QWidget, store *config.Store, symbol string, current float64) {
	dialog, body := newCardDialog(parent, fmt.Sprintf(constants.FmtTitleAlert, symbol))
	dialog.Resize(int(constants.AlertDialogWidth), int(constants.AlertDialogHeight))

	currentLabel := qt.NewQLabel5(fmt.Sprintf(constants.FmtAlertCurrent, current), dialog.QWidget)
	currentLabel.SetStyleSheet("background: transparent;")
	body.AddWidget(currentLabel.QWidget)

	priceLabel := qt.NewQLabel5(constants.LabelAlertPrice, dialog.QWidget)
	priceLabel.SetStyleSheet("background: transparent; color: " + cssRGB(constants.ColorNeutral) + ";")
	body.AddWidget(priceLabel.QWidget)
	priceEdit := qt.NewQLineEdit4("", dialog.QWidget)
	priceEdit.SetStyleSheet(inputStyle())
	body.AddWidget(priceEdit.QWidget)

	// Live distance of the entered price from the current one, colored like
	// a price change; kept laid out (empty when invalid) for a stable form.
	deltaLabel := qt.NewQLabel(dialog.QWidget)
	deltaLabel.SetStyleSheet("background: transparent;")
	body.AddWidget(deltaLabel.QWidget)

	errorLabel := qt.NewQLabel(dialog.QWidget)
	errorLabel.SetStyleSheet("background: transparent; color: " + cssRGB(constants.ColorDown) + ";")
	body.AddWidget(errorLabel.QWidget)
	body.AddStretch()

	priceEdit.OnTextChanged(func(text string) {
		errorLabel.SetText("")
		value, ok := parseAlertPrice(text)
		if !ok || current <= 0 {
			deltaLabel.SetText("")
			return
		}
		percent := (value - current) / current * constants.PercentMax
		deltaColor, sign := changeStyle(percent)
		deltaLabel.SetText(fmt.Sprintf(constants.FmtAlertDelta, sign, percent))
		deltaLabel.SetStyleSheet("background: transparent; color: " + cssRGB(deltaColor) + ";")
	})

	actions := qt.NewQHBoxLayout2()
	actions.AddStretch()
	cancelButton := qt.NewQPushButton5(constants.LabelCancel, dialog.QWidget)
	cancelButton.SetStyleSheet(dialogButtonStyle(false))
	cancelButton.SetCursor(qt.NewQCursor2(qt.PointingHandCursor))
	cancelButton.OnClicked(func() { dialog.Reject() })
	actions.AddWidget(cancelButton.QWidget)
	addButton := qt.NewQPushButton5(constants.LabelAddAlert, dialog.QWidget)
	addButton.SetStyleSheet(dialogButtonStyle(true))
	addButton.SetCursor(qt.NewQCursor2(qt.PointingHandCursor))
	addButton.OnClicked(func() {
		value, ok := parseAlertPrice(priceEdit.Text())
		if !ok {
			errorLabel.SetText(constants.MsgErrAlertPrice)
			return
		}
		if value == current {
			errorLabel.SetText(constants.MsgErrAlertSame)
			return
		}
		for _, existing := range store.Get().Alerts {
			if existing.Symbol == symbol && existing.Price == value {
				errorLabel.SetText(constants.MsgErrAlertDup)
				return
			}
		}
		alert := notify.Alert{Symbol: symbol, Price: value, Above: value > current}
		if err := store.Update(func(conf *config.Config) {
			conf.Alerts = append(conf.Alerts, alert)
		}); err != nil {
			errorLabel.SetText(err.Error())
			return
		}
		dialog.Accept()
	})
	actions.AddWidget(addButton.QWidget)
	body.AddLayout(actions.QLayout)

	dialog.Exec()
}
