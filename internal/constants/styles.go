package constants

// --- Qt stylesheet templates ---
//
// The user interface is styled by internal/qtui/ui/theme.qss, which the app
// installs application-wide. Only styling the theme cannot know — values
// that come from the user's config — is built in Go, and its template lives
// here.

const (
	// StyleWindowCard paints the main window card from (background rgba,
	// corner radius px). It is the one stylesheet built in Go: its color and
	// opacity come from the user's config. Everything else lives in the
	// theme (internal/qtui/ui/theme.qss).
	StyleWindowCard = "#card { background-color: %s; border-radius: %dpx; }"
)
