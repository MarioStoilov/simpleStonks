package qtui

import (
	qt "github.com/mappu/miqt/qt6"

	"github.com/MarioStoilov/simplestonks/internal/constants"
)

// newCardDialog builds a frameless card-styled modal dialog from the
// CardDialog form (dialogcard.ui): a rounded opaque card with a title row
// and a ✕ close button, dragged by its title. The returned layout receives
// the dialog's content; Esc rejects as usual.
func newCardDialog(parent *qt.QWidget, title string) (*qt.QDialog, *qt.QVBoxLayout) {
	dialog := qt.NewQDialog(parent)
	dialog.SetWindowTitle(title)
	dialog.SetModal(true)
	dialog.SetWindowFlags(qt.Dialog | qt.FramelessWindowHint)
	dialog.SetAttribute(qt.WA_TranslucentBackground)

	outer := qt.NewQVBoxLayout(dialog.QWidget)
	outer.SetContentsMargins(0, 0, 0, 0)

	loaded := loadForm(cardDialogForm)
	card := loaded.frame("dialogCard")
	// Each frameless dialog carries the theme on its card, for the same
	// reason the main window does (see App.applyBackgroundStyle).
	card.SetStyleSheet(themeStyleSheet)
	outer.AddWidget(card.QWidget)

	titleLabel := loaded.label("titleLabel")
	titleLabel.SetText(title)
	// Dragging is confined to the title label: a whole-dialog drag handler
	// would swallow presses meant for interactive content (e.g. the About
	// dialog\'s links, whose activation fires on release).
	titleLabel.OnMousePressEvent(func(super func(event *qt.QMouseEvent), event *qt.QMouseEvent) {
		dialog.WindowHandle().StartSystemMove()
	})
	loaded.button("dialogCloseButton").OnClicked(func() { dialog.Reject() })

	return dialog, loaded.vbox("body")
}

// showConfirmDialog runs a modal confirmation (confirm.ui) with Cancel and
// a labeled confirm action, reporting whether the user confirmed.
func showConfirmDialog(parent *qt.QWidget, title, message, confirmLabel string) bool {
	dialog, body := newCardDialog(parent, title)
	dialog.SetFixedWidth(int(constants.ConfirmDialogWidth))
	loaded := loadForm(confirmForm)
	body.AddWidget(loaded.root)
	body.SetStretchFactor(loaded.root, 1)

	loaded.label("messageLabel").SetText(message)
	confirmButton := loaded.button("confirmButton")
	confirmButton.SetText(confirmLabel)
	confirmButton.OnClicked(func() { dialog.Accept() })
	loaded.button("cancelButton").OnClicked(func() { dialog.Reject() })
	return dialog.Exec() == int(qt.QDialog__Accepted)
}
