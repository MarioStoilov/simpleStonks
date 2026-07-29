package constants

import "image/color"

// --- UI geometry & typography ---

const (
	// MainWindowWidth/Height is the initial size of the main window.
	MainWindowWidth  = 1000
	MainWindowHeight = 640

	// SettingsWindowWidth/Height is the size of the settings window.
	SettingsWindowWidth  = 600
	SettingsWindowHeight = 560

	// SearchDialogWidth/Height is the size of the add-symbol search dialog.
	SearchDialogWidth  = 500
	SearchDialogHeight = 440

	// SearchScrollMinWidth/Height is the minimum size of the result list.
	SearchScrollMinWidth  = 440
	SearchScrollMinHeight = 280

	// PreviewDialogWidth/Height is the size of the search-result preview.
	PreviewDialogWidth  = 520
	PreviewDialogHeight = 440

	// AlertDialogWidth/Height is the size of the price-alert dialog.
	AlertDialogWidth  = 400
	AlertDialogHeight = 260

	// ConfirmDialogWidth is the width of confirmation modals (the height
	// follows the wrapped message).
	ConfirmDialogWidth = 420

	// AboutDialogWidth is the width of the About dialog.
	AboutDialogWidth = 460

	// AboutLogoSize is the rendered size of the logo in the About dialog.
	AboutLogoSize = 72

	// GridCellWidth/Height is the home-grid cell size; edit mode is taller
	// to fit the reorder/remove control row.
	GridCellWidth      = 300
	GridCellHeight     = 240
	GridCellEditHeight = 280

	// SidebarMinWidth is the minimum width of the detail-view sidebar.
	SidebarMinWidth = 190

	// ChartMinWidth/Height is the minimum chart size (mini tiles).
	ChartMinWidth  = 120
	ChartMinHeight = 80

	// SwatchWidth/Height is the size of the background color swatch.
	SwatchWidth  = 48
	SwatchHeight = 28

	// SwatchDisabledAlpha dims a disabled color swatch's color.
	SwatchDisabledAlpha = 0x40

	// HeaderIconSize is the rendered size of the SVG header icons (the
	// alert bell, the offline indicator).
	HeaderIconSize = 18

	// DisabledAddWidth/Height is the size of the disabled Add stand-in.
	DisabledAddWidth  = 72
	DisabledAddHeight = 32

	// TooltipOffset shifts a hover tooltip away from the pointer.
	TooltipOffset = 10

	// ResizeGripMargin is how many pixels of a frameless window's edge act
	// as an invisible resize grip.
	ResizeGripMargin = 8

	// SidebarTileHeight is the minimum height of the compact tiles in the
	// detail view's sidebar.
	SidebarTileHeight = 150

	// ScrollBarWidth/ScrollBarMinHandle size the slim overlay scrollbars.
	ScrollBarWidth     = 8
	ScrollBarMinHandle = 24

	// ChartPadding insets the plot area inside the chart widget.
	ChartPadding = 4

	// AxisTextSize is the font size of the axis labels.
	AxisTextSize = 12

	// AxisGap separates axis labels from the plot area.
	AxisGap = 4

	// XTickSpacing is the minimum horizontal pixels per time label.
	XTickSpacing = 80

	// YTickSpacing is the minimum vertical pixels per price label.
	YTickSpacing = 48

	// MaxYTicks caps the number of y-axis reference values.
	MaxYTicks = 8

	// DashLen/DashGap shape the dashed reference and guide lines.
	DashLen = 4
	DashGap = 4

	// DotRadius is the radius of the hover marker dot.
	DotRadius = 3.5

	// TipPad is the padding inside the hover tooltip.
	TipPad = 4

	// TipGap separates the hover dot from its tooltip.
	TipGap = 8

	// ChartLineWidth is the stroke width of the price line.
	ChartLineWidth = 1.5

	// HairlineWidth is the stroke width of guides, dashes, and borders.
	HairlineWidth = 1

	// NameTextSize is the font size of the small friendly-name line shown
	// under a symbol (on tiles and the detail header).
	NameTextSize = 11

	// FlashAlpha is the starting opacity of the price flash background.
	FlashAlpha = 0x66

	// AfterHoursDimAlpha is the line opacity of the after-hours chart
	// segment, dimmed against the regular-session line.
	AfterHoursDimAlpha = 0x59

	// FlashPad is the extra flash width to the left of the price number.
	FlashPad = 3

	// TileCornerRadius rounds the home-grid/sidebar tile cards.
	TileCornerRadius = 6

	// PanelCornerRadius rounds small panels: tooltips, swatches, rows.
	PanelCornerRadius = 4

	// FlashCornerRadius rounds the price flash background.
	FlashCornerRadius = 3
)

// --- Colors (dark palette; placeholder styling while design is finalized) ---

var (
	// ColorUp/ColorDown/ColorNeutral color rising/falling/unchanged prices.
	ColorUp      = color.NRGBA{R: 0x26, G: 0xa6, B: 0x5b, A: 0xff}
	ColorDown    = color.NRGBA{R: 0xd0, G: 0x3a, B: 0x3a, A: 0xff}
	ColorNeutral = color.NRGBA{R: 0x8a, G: 0x8a, B: 0x8a, A: 0xff}

	// ColorCardBg/ColorSelected/ColorHover style tiles and hover states.
	ColorCardBg   = color.NRGBA{R: 0x24, G: 0x26, B: 0x2e, A: 0xff}
	ColorSelected = color.NRGBA{R: 0x30, G: 0x3a, B: 0x52, A: 0xff}
	ColorHover    = color.NRGBA{R: 0x2c, G: 0x30, B: 0x3c, A: 0xff}

	// ColorAxis colors chart axis labels and reference lines.
	ColorAxis = color.NRGBA{R: 0x6e, G: 0x72, B: 0x7e, A: 0xff}

	// ColorForeground is the default text color on the dark palette.
	ColorForeground = color.NRGBA{R: 0xe6, G: 0xe6, B: 0xe6, A: 0xff}

	// ColorChartBg is the chart plot background.
	ColorChartBg = color.NRGBA{R: 0x1e, G: 0x1e, B: 0x24, A: 0xff}

	// ColorDisabledBg/ColorDisabledFg style the disabled Add stand-in.
	ColorDisabledBg = color.NRGBA{R: 0x3a, G: 0x3a, B: 0x40, A: 0xff}
	ColorDisabledFg = color.NRGBA{R: 0x80, G: 0x80, B: 0x80, A: 0xff}
)
