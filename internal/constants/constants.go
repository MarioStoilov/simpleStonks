// Package constants centralizes the magic numbers and strings used across
// simpleStonks, grouped by domain (app identity, storage, network, timing,
// UI geometry, colors, formats, and user-facing text). Typed enum values
// (model.Range, config.LogLevel), struct tags, protocol mapping tables
// (yahooParams), and purely diagnostic log messages stay at their definition
// sites.
package constants

import "time"

// --- App identity ---

const (
	// AppName is the human-facing application name.
	AppName = "simpleStonks"

	// AppID is the Fyne application ID; it also namespaces Fyne's own storage.
	AppID = "com.github.mariostoilov.simplestonks"

	// AppVersion is the released application version, shown in the About
	// dialog. Keep it in sync with the `version` key in snap/snapcraft.yaml.
	AppVersion = "1.4.0"

	// RepoURL is the public home of the project, linked from the About dialog.
	RepoURL = "https://github.com/MarioStoilov/simpleStonks"

	// LicenseURL points at the project license, linked from the About dialog.
	LicenseURL = RepoURL + "/blob/main/LICENSE"
)

// --- Storage (config, logs, permissions) ---

const (
	// AppDirName is the per-app subdirectory under the XDG config/state dirs.
	AppDirName = "simplestonks"

	// ConfigFileName is the config file name inside AppDirName.
	ConfigFileName = "config.json"

	// LogFileName is the default log file name inside AppDirName.
	LogFileName = "simplestonks.log"

	// TmpFileSuffix is appended to a file path for atomic write-then-rename.
	TmpFileSuffix = ".tmp"

	// EnvXDGStateHome is the environment variable overriding the state dir.
	EnvXDGStateHome = "XDG_STATE_HOME"

	// StateSubdir is the XDG state directory fallback under $HOME.
	StateSubdir = ".local/state"

	// DirPerm and FilePerm are the modes for created directories and files.
	DirPerm  = 0o755
	FilePerm = 0o644

	// DefaultLogMaxSizeMB is the default log rotation threshold.
	DefaultLogMaxSizeMB = 5

	// DefaultLogMaxArchives is the default number of rotated logs retained.
	DefaultLogMaxArchives = 3

	// DefaultBackgroundColor and DefaultBackgroundOpacity match Fyne's
	// dark-theme window background at full opacity.
	DefaultBackgroundColor   = "#171718"
	DefaultBackgroundOpacity = 1.0

	// DefaultChartBackground is the default chart plot background color
	// (the hex form of ColorChartBg — keep them in sync).
	DefaultChartBackground = "#1e1e24"

	// DefaultChartGridSize/MinChartGridSize size the chart grid squares in
	// pixels; the minimum keeps the grid drawable and the paint loop sane.
	DefaultChartGridSize = 24
	MinChartGridSize     = 4

	// DefaultChartGridColor is the default chart grid line color, a shade
	// above the plot background.
	DefaultChartGridColor = "#2a2a33"

	// DefaultChartFillOpacity is the default opacity of the chart area fill,
	// matching the app logo's fill.
	DefaultChartFillOpacity = 0.25

	// BytesPerMiB converts the config's MB numbers into bytes.
	BytesPerMiB = 1024 * 1024
)

// DefaultSymbols is the tracked list seeded on first run. Callers must copy
// it before mutating.
var DefaultSymbols = []string{"^GSPC", "^IXIC", "AAPL"}

// --- Network (Yahoo Finance endpoints) ---

const (
	// YahooChartBaseURL is the keyless Yahoo Finance chart endpoint.
	YahooChartBaseURL = "https://query1.finance.yahoo.com/v8/finance/chart/"

	// YahooSearchURL is the keyless Yahoo Finance search endpoint.
	YahooSearchURL = "https://query1.finance.yahoo.com/v1/finance/search"

	// YahooUserAgent is sent with every request; Yahoo tends to reject
	// requests carrying Go's default user agent.
	YahooUserAgent = "simplestonks/0.1 (+https://github.com/MarioStoilov/simpleStonks)"

	// YahooSearchQuotesCount and YahooSearchNewsCount bound the search
	// response (we only consume quotes).
	YahooSearchQuotesCount = "10"
	YahooSearchNewsCount   = "0"

	// HTTPClientTimeout bounds a whole HTTP request at the client level.
	HTTPClientTimeout = 10 * time.Second

	// FetchTimeout bounds a single provider fetch (context deadline).
	FetchTimeout = 15 * time.Second
)

// --- Desktop notifications (freedesktop D-Bus) ---

const (
	// NotifyDBusService/Path/Method address the freedesktop notification
	// service on the session bus (reachable under snap confinement through
	// the desktop interface).
	NotifyDBusService = "org.freedesktop.Notifications"
	NotifyDBusPath    = "/org/freedesktop/Notifications"
	NotifyDBusMethod  = "org.freedesktop.Notifications.Notify"

	// NotifyExpireDefault lets the desktop pick the notification timeout.
	NotifyExpireDefault = -1

	// NotifyIconFileName is where (under the user cache dir + AppDirName)
	// the embedded app logo is materialized so the notification daemon —
	// which runs outside any snap confinement — can read it for the
	// notification icon.
	NotifyIconFileName = "notify-icon.svg"

	// NotifyActionDefault is the reserved freedesktop action key for a
	// click on the notification body itself.
	NotifyActionDefault = "default"

	// NotifyMemberActionInvoked/Closed are the notification service's
	// signal member names (a clicked action; a dismissed notification);
	// NotifySignalActionInvoked/Closed their fully qualified names.
	NotifyMemberActionInvoked = "ActionInvoked"
	NotifyMemberClosed        = "NotificationClosed"
	NotifySignalActionInvoked = NotifyDBusService + "." + NotifyMemberActionInvoked
	NotifySignalClosed        = NotifyDBusService + "." + NotifyMemberClosed

	// NotifySignalBuffer bounds the queued notification signals.
	NotifySignalBuffer = 16

	// NotifyQuick/Moderate/LongDuration are the on-screen times behind the
	// notification-duration setting.
	NotifyQuickDuration    = 3 * time.Second
	NotifyModerateDuration = 7 * time.Second
	NotifyLongDuration     = 15 * time.Second
)

// --- Timing ---

const (
	// DefaultRefreshInterval is the live-tick polling cadence used both as
	// the config default and as the fallback for non-positive values.
	DefaultRefreshInterval = 30 * time.Second

	// ConfigReloadDebounce coalesces filesystem events before reloading the
	// config; editors and atomic saves emit several events per write.
	ConfigReloadDebounce = 150 * time.Millisecond

	// SearchDebounce is how long to wait after the last keystroke before
	// querying the search endpoint.
	SearchDebounce = 300 * time.Millisecond

	// FlashDuration is how long a price-update flash takes to fade out.
	FlashDuration = 900 * time.Millisecond

	// SessionShiftDay is the whole-day step used to translate the upcoming
	// session's windows back onto the fetched candles' day while the market
	// is closed (Yahoo reports the next session's windows overnight).
	SessionShiftDay = 24 * time.Hour

	// SessionShiftMaxDays bounds that backward search; it covers weekends
	// and holiday breaks while stopping runaway loops on garbage timestamps.
	SessionShiftMaxDays = 7
)

// --- Formats (time layouts, numbers) ---

const (
	// TimeFmtClock renders an intraday time, e.g. "15:04".
	TimeFmtClock = "15:04"

	// TimeFmtWeekdayDay renders a weekday + day, e.g. "Mon 02".
	TimeFmtWeekdayDay = "Mon 02"

	// TimeFmtDayMonth renders a day + month, e.g. "02 Jan".
	TimeFmtDayMonth = "02 Jan"

	// TimeFmtMonth renders a month, e.g. "Jan".
	TimeFmtMonth = "Jan"

	// TimeFmtYear renders a year, e.g. "2006".
	TimeFmtYear = "2006"

	// TimeFmtWeekdayDate renders a weekday + date, e.g. "Mon, 02 Jan".
	TimeFmtWeekdayDate = "Mon, 02 Jan"

	// TimeFmtFullDate renders a full date, e.g. "02 Jan 2006".
	TimeFmtFullDate = "02 Jan 2006"

	// FmtPrice renders a price value, e.g. "123.45".
	FmtPrice = "%.2f"

	// FmtPercentChange renders a signed percent change, e.g. "+1.23%".
	FmtPercentChange = "%s%.2f%%"

	// FmtPriceChange renders an absolute + percent change pair,
	// e.g. "+1.23 (+0.45%)".
	FmtPriceChange = "%s%.2f (%s%.2f%%)"

	// FmtExtendedQuote renders an extended-hours quote from
	// (prefix, price, sign, percent), e.g. "Pre-market: 185.05 (+0.85%)".
	FmtExtendedQuote = "%s: %.2f (%s%.2f%%)"

	// FmtLogArchive renders a rotated log archive name from (path, index).
	FmtLogArchive = "%s.%d"

	// PercentMax converts between a 0..1 fraction and a 0..100 percentage.
	PercentMax = 100

	// AlertPriceDecimals is how many digits an alert price may carry after
	// the decimal dot.
	AlertPriceDecimals = 2
)
