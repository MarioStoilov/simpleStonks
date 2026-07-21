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
	SectionGeneral    = "General"
	SectionAppearance = "Appearance"
	SectionLogging    = "Logging"

	// Settings form labels.
	LabelDefaultRange      = "Default range"
	LabelRefreshInterval   = "Refresh interval (s)"
	LabelExtendedHours     = "Extended hours (pre/post market)"
	LabelBackgroundColor   = "Background color"
	LabelBackgroundOpacity = "Background opacity (%)"
	LabelLogLevel          = "Log level"
	LabelLogFile           = "Log file (blank = default)"
	LabelLogMaxSize        = "Log max size (MB)"
	LabelLogArchives       = "Log archives kept"

	// Settings validation messages (shown in error dialogs).
	MsgErrRefreshInterval = "refresh interval must be a whole number of seconds ≥ 1"
	MsgErrLogMaxSize      = "log max size must be a non-negative whole number of MB"
	MsgErrLogArchives     = "log archives kept must be a non-negative whole number"

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

	// Extended-hours (pre-market / after-hours) display on the detail view.
	LabelExtendedToggle = "Pre/post market data"
	TipExtendedHours    = "Show pre-market and after-hours prices and charting on the 1D range"
	LabelPreMarket      = "Pre-market"
	LabelAfterHours     = "After-hours"

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
