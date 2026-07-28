package qtui

import (
	"fmt"

	qt "github.com/mappu/miqt/qt6"

	"github.com/MarioStoilov/simplestonks/internal/constants"
)

// newCardDialog builds a frameless card-styled modal dialog matching the
// widget look: rounded opaque card, a title row with a ✕ close button, and
// drag-to-move on the background. The returned layout receives the dialog's
// content; Esc rejects as usual.
func newCardDialog(parent *qt.QWidget, title string) (*qt.QDialog, *qt.QVBoxLayout) {
	dialog := qt.NewQDialog(parent)
	dialog.SetWindowTitle(title)
	dialog.SetModal(true)
	dialog.SetWindowFlags(qt.Dialog | qt.FramelessWindowHint)
	dialog.SetAttribute(qt.WA_TranslucentBackground)

	outer := qt.NewQVBoxLayout(dialog.QWidget)
	outer.SetContentsMargins(0, 0, 0, 0)

	card := qt.NewQFrame(dialog.QWidget)
	card.SetObjectName(*qt.NewQAnyStringView3("dialogCard"))
	card.SetStyleSheet(fmt.Sprintf(
		"#dialogCard { background-color: %s; border-radius: %dpx; } QLabel { color: %s; }",
		cssRGB(constants.ColorCardBg), int(constants.TileCornerRadius), cssRGB(constants.ColorForeground)))
	outer.AddWidget(card.QWidget)

	body := qt.NewQVBoxLayout(card.QWidget)

	bar := qt.NewQHBoxLayout2()
	titleLabel := qt.NewQLabel5(title, card.QWidget)
	titleLabel.SetStyleSheet("background: transparent; font-weight: 600;")
	// Dragging is confined to the title label: a whole-dialog drag handler
	// would swallow presses meant for interactive content (e.g. the About
	// dialog's links, whose activation fires on release).
	titleLabel.OnMousePressEvent(func(super func(event *qt.QMouseEvent), event *qt.QMouseEvent) {
		dialog.WindowHandle().StartSystemMove()
	})
	bar.AddWidget(titleLabel.QWidget)
	bar.AddStretch()
	closeButton := qt.NewQPushButton5("✕", card.QWidget)
	closeButton.SetStyleSheet(windowButtonStyle(cssRGB(constants.ColorDown)))
	closeButton.SetCursor(qt.NewQCursor2(qt.PointingHandCursor))
	closeButton.OnClicked(func() { dialog.Reject() })
	bar.AddWidget(closeButton.QWidget)
	body.AddLayout(bar.QLayout)

	return dialog, body
}

// dialogButtonStyle styles a dialog action button; primary marks the default
// action with the selection color.
func dialogButtonStyle(primary bool) string {
	background := constants.ColorHover
	if primary {
		background = constants.ColorSelected
	}
	return fmt.Sprintf(
		"QPushButton { background-color: %s; color: %s; border: none; border-radius: %dpx; padding: 5px 14px; }"+
			" QPushButton:hover { background-color: %s; }"+
			" QPushButton:disabled { background-color: %s; color: %s; }",
		cssRGB(background), cssRGB(constants.ColorForeground), int(constants.PanelCornerRadius),
		cssRGB(constants.ColorSelected), cssRGB(constants.ColorDisabledBg), cssRGB(constants.ColorDisabledFg))
}

// checkBoxStyle styles a checkbox for the dark card, matching inputStyle's
// palette: a bordered indicator that, when on, fills with the selection
// color around a bright center dot so the state reads at a glance.
func checkBoxStyle() string {
	return fmt.Sprintf(
		"QCheckBox { background: transparent; color: %s; spacing: 8px; }"+
			" QCheckBox::indicator { width: 14px; height: 14px; border: 1px solid %s; border-radius: %dpx; background-color: %s; }"+
			" QCheckBox::indicator:checked { background-color: qradialgradient(cx: 0.5, cy: 0.5, radius: 0.5, fx: 0.5, fy: 0.5, stop: 0.45 %s, stop: 0.6 %s); }",
		cssRGB(constants.ColorForeground), cssRGB(constants.ColorAxis),
		int(constants.PanelCornerRadius), cssRGB(constants.ColorChartBg),
		cssRGB(constants.ColorForeground), cssRGB(constants.ColorSelected))
}

// inputStyle styles line edits and combo boxes for the dark card.
func inputStyle() string {
	return fmt.Sprintf(
		"QLineEdit, QComboBox { background-color: %s; color: %s; border: 1px solid %s; border-radius: %dpx; padding: 5px 8px; }"+
			" QLineEdit:disabled, QComboBox:disabled { background-color: %s; color: %s; border-color: %s; }"+
			" QComboBox QAbstractItemView { background-color: %s; color: %s; selection-background-color: %s; }",
		cssRGB(constants.ColorChartBg), cssRGB(constants.ColorForeground), cssRGB(constants.ColorAxis),
		int(constants.PanelCornerRadius),
		cssRGB(constants.ColorDisabledBg), cssRGB(constants.ColorDisabledFg), cssRGB(constants.ColorDisabledBg),
		cssRGB(constants.ColorCardBg), cssRGB(constants.ColorForeground), cssRGB(constants.ColorSelected))
}
