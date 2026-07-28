package constants

// --- Qt stylesheet templates ---
//
// Multi-rule Qt stylesheets live here as fmt.Sprintf templates, filled in
// with palette colors and metrics at their use sites. Trivial one-rule
// inline styles (e.g. "background: transparent;") stay at theirs.

const (
	// StyleWindowCard paints the main window card from (background rgba,
	// corner radius px, label color).
	StyleWindowCard = "#card { background-color: %s; border-radius: %dpx; } QLabel { color: %s; }"

	// StyleWindowButton styles a window-control button from (text color,
	// corner radius px, hover background, hover text color).
	StyleWindowButton = "QPushButton { background: transparent; color: %s; border: none; border-radius: %dpx; padding: 4px 10px; }" +
		" QPushButton:hover { background-color: %s; color: %s; }"

	// StyleDialogCard paints a card dialog from (background color, corner
	// radius px, label color).
	StyleDialogCard = "#dialogCard { background-color: %s; border-radius: %dpx; } QLabel { color: %s; }"

	// StyleDialogButton styles a dialog action button from (background,
	// text color, corner radius px, hover background, disabled background,
	// disabled text color).
	StyleDialogButton = "QPushButton { background-color: %s; color: %s; border: none; border-radius: %dpx; padding: 5px 14px; }" +
		" QPushButton:hover { background-color: %s; }" +
		" QPushButton:disabled { background-color: %s; color: %s; }"

	// StyleToggleButton styles a pill toggle button from (background, text
	// color, corner radius px, hover background).
	StyleToggleButton = "QPushButton { background-color: %s; color: %s; border: none; border-radius: %dpx; padding: 4px 10px; }" +
		" QPushButton:hover { background-color: %s; }"

	// StyleCheckBox styles a checkbox from (text color, indicator border
	// color, corner radius px, indicator background, checked dot color,
	// checked fill color).
	StyleCheckBox = "QCheckBox { background: transparent; color: %s; spacing: 8px; }" +
		" QCheckBox::indicator { width: 14px; height: 14px; border: 1px solid %s; border-radius: %dpx; background-color: %s; }" +
		" QCheckBox::indicator:checked { background-color: qradialgradient(cx: 0.5, cy: 0.5, radius: 0.5, fx: 0.5, fy: 0.5, stop: 0.45 %s, stop: 0.6 %s); }"

	// StyleInput styles line edits and combo boxes from (background, text
	// color, border color, corner radius px, disabled background, disabled
	// text color, disabled border color, popup background, popup text
	// color, popup selection background).
	StyleInput = "QLineEdit, QComboBox { background-color: %s; color: %s; border: 1px solid %s; border-radius: %dpx; padding: 5px 8px; }" +
		" QLineEdit:disabled, QComboBox:disabled { background-color: %s; color: %s; border-color: %s; }" +
		" QComboBox QAbstractItemView { background-color: %s; color: %s; selection-background-color: %s; }"

	// StyleFieldLabel styles a settings field label from (text color,
	// disabled text rgba).
	StyleFieldLabel = "QLabel { background: transparent; color: %s; } QLabel:disabled { color: %s; }"

	// StyleSwatch styles a color-picker swatch button from (color, border
	// color, corner radius px, disabled color rgba, disabled border color).
	StyleSwatch = "QPushButton { background-color: %s; border: 1px solid %s; border-radius: %dpx; }" +
		" QPushButton:disabled { background-color: %s; border-color: %s; }"

	// StyleTile styles a symbol tile from (background, corner radius px,
	// hover background).
	StyleTile = "#tile { background-color: %s; border-radius: %dpx; } #tile:hover { background-color: %s; }"

	// StyleAlertPill styles a pending-alert pill from (background, corner
	// radius px).
	StyleAlertPill = "#pill { background-color: %s; border-radius: %dpx; }"

	// StylePriceFlash backs a price label during a change flash from (base
	// style, flash rgba, corner radius px).
	StylePriceFlash = "%s background-color: %s; border-radius: %dpx;"

	// StyleSmallText renders secondary text from (text color, font size px).
	StyleSmallText = "background: transparent; color: %s; font-size: %dpx;"

	// StyleSearchResults styles the search-result list from (item corner
	// radius px, hover/selection background).
	StyleSearchResults = "QListWidget { background: transparent; border: none; }" +
		" QListWidget::item { border-radius: %dpx; }" +
		" QListWidget::item:hover, QListWidget::item:selected { background-color: %s; }"

	// StyleScrollArea styles transparent scroll areas with slim dark
	// scrollbars from (bar width px, bar height px, handle color, handle
	// corner radius px, handle min height px, handle hover color).
	StyleScrollArea = "QScrollArea { background: transparent; border: none; }" +
		" QScrollBar { background: transparent; }" +
		" QScrollBar:vertical { width: %dpx; }" +
		" QScrollBar:horizontal { height: %dpx; }" +
		" QScrollBar::handle { background: %s; border-radius: %dpx; }" +
		" QScrollBar::handle:vertical { min-height: %dpx; }" +
		" QScrollBar::handle:hover { background: %s; }" +
		" QScrollBar::add-line, QScrollBar::sub-line { width: 0; height: 0; }" +
		" QScrollBar::add-page, QScrollBar::sub-page { background: transparent; }"
)
