package constants

// --- User-facing text (labels, titles, messages) ---

const (
	// Window & dialog titles.
	TitleSettings    = AppName + " — Settings"
	TitleAbout       = "About " + AppName
	TitleAddIndex    = "Add index"
	TitlePreview     = "Preview"
	TitleColorPicker = "Background color"

	// Button labels.
	LabelEdit   = "Edit"
	LabelDone   = "Done"
	LabelAdd    = "Add"
	LabelSave   = "Save"
	LabelCancel = "Cancel"
	LabelClose  = "Close"
	LabelChoose = "Choose…"

	// Settings sections.
	SectionGeneral       = "General"
	SectionAppearance    = "Appearance"
	SectionNotifications = "Notifications"
	SectionLogging       = "Logging"

	// Settings form labels.
	LabelDefaultRange      = "Default range"
	LabelRefreshInterval   = "Refresh interval (s)"
	LabelExtendedHours     = "Extended hours (pre/post market)"
	LabelBackgroundColor   = "Background color"
	LabelBackgroundOpacity = "Background opacity (%)"
	LabelChartBackground   = "Chart background color"
	LabelChartGrid         = "Checkered chart grid"
	LabelChartGridSize     = "Grid square size (px)"
	LabelChartGridColor    = "Grid line color"
	LabelChartFill         = "Chart area fill (up/down shading)"
	LabelChartFillOpacity  = "Fill opacity (%)"
	TipChartGrid           = "Draw a checkered graph-paper grid behind the charts"
	TipChartFill           = "Shade the area between the price line and the dashed previous-close reference — green above the line, red below"
	LabelLogLevel          = "Log level"
	LabelLogFile           = "Log file (blank = default)"
	LabelLogMaxSize        = "Log max size (MB)"
	LabelLogArchives       = "Log archives kept"
	TitleChartColorPicker  = "Chart background color"
	TitleGridColorPicker   = "Grid line color"

	// Settings validation messages (shown in error dialogs).
	MsgErrRefreshInterval = "refresh interval must be a whole number of seconds ≥ 1"
	MsgErrLogMaxSize      = "log max size must be a non-negative whole number of MB"
	MsgErrLogArchives     = "log archives kept must be a non-negative whole number"
	MsgErrGridSize        = "grid square size must be a whole number of pixels ≥ 4"

	// Search dialog.
	PlaceholderSearch = "Search name or symbol — e.g. Apple or AAPL"
	MsgSearchPrompt   = "Start typing to search…"
	MsgSearching      = "Searching…"
	MsgSearchFailed   = "Search failed (offline?)"
	MsgNoMatches      = "No matches"
	FmtResultCount    = "%d result(s)"
	TipAlreadyTracked = "Already tracking this index"

	// Prices & tiles.
	PricePlaceholder = "—"
	MsgUnavailable   = "unavailable"

	// Offline indicator (window header, assets.OfflineSVG).
	TipOffline = "Network connection lost — showing the last loaded data"

	// Extended-hours (pre-market / after-hours) display on the detail view.
	LabelExtendedToggle = "Pre/post market data"
	TipExtendedHours    = "Show pre-market and after-hours prices and charting on the 1D range"
	LabelPreMarket      = "Pre-market"
	LabelAfterHours     = "After-hours"

	// Price alerts: the detail-view bell dialog, the pending-alert pills
	// under the chart, and the fired-alert desktop notification. (The bell
	// button itself renders the embedded assets.BellPlusSVG.)
	TipSetAlert      = "Set a price alert"
	FmtTitleAlert    = "Price alert — %s"
	FmtAlertCurrent  = "Current price: %.2f"
	LabelAlertPrice  = "Alert price"
	LabelAddAlert    = "Add alert"
	FmtAlertDelta    = "%s%.2f%% vs current"
	MsgErrAlertPrice = "alert price must be a positive number with up to 2 decimals"
	MsgErrAlertSame  = "alert price must differ from the current price"
	MsgErrAlertDup   = "an identical alert already exists"
	SymbolAlertUp    = "▲"
	SymbolAlertDown  = "▼"
	FmtAlertPill     = "%s %.2f"
	FmtAlertSummary  = "%s price alert"
	MsgAlertRose     = "Rose above"
	MsgAlertFell     = "Fell below"
	FmtAlertBody     = "%s %.2f — now %.2f"
	LabelNotifyOpen  = "Open"

	// Notification settings (Settings → Notifications).
	LabelNotifications    = "Price alert notifications"
	TipNotifications      = "Fire desktop notifications when a price alert is hit; hides the alert bell and the pending alerts when off"
	LabelNotifyDuration   = "Notification duration"
	LabelClearAlerts      = "Clear all price alerts"
	TitleClearAlerts      = "Clear all price alerts"
	LabelClear            = "Clear"
	MsgConfirmClearAlerts = "Are you sure you want to clear all price alerts?\n\n" +
		"Please note that this action is not reversible and you will be " +
		"required to recreate any alert you have removed"

	// Separators used when joining symbol/name/market fragments.
	SepTitle = "  ·  "
	SepMeta  = " · "

	// About dialog copy.
	LabelVersionPrefix = "Version "
	LabelSourceLink    = "Source on GitHub"
	LabelLicenseLink   = "MIT license"
	AboutDescription   = "A bare-bones, Yahoo-Finance-style stock tracker —\nvibe coded out of boredom."
	AboutDisclaimer    = "Market data comes from unofficial, keyless Yahoo Finance " +
		"endpoints and can lag, break, or disappear. Nothing here is financial advice."
)
