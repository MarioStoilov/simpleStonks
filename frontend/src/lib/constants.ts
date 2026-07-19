// Frontend constants mirroring internal/constants (ui.go) so the web UI
// matches the palette and chart metrics of the original Fyne UI. Keep the two
// in sync when tweaking either.

export const COLOR_UP = '#26a65b';
export const COLOR_DOWN = '#d03a3a';
export const COLOR_NEUTRAL = '#8a8a8a';
export const COLOR_CARD_BG = '#24262e';
export const COLOR_SELECTED = '#303a52';
export const COLOR_HOVER = '#2c303c';
export const COLOR_AXIS = '#6e727e';
export const COLOR_CHART_BG = '#1e1e24';
export const COLOR_FOREGROUND = '#e6e6e6';

export const AXIS_TEXT_SIZE = 12;
export const AXIS_GAP = 4;
export const CHART_PADDING = 4;
export const X_TICK_SPACING = 80;
export const Y_TICK_SPACING = 48;
export const MAX_Y_TICKS = 8;
export const DOT_RADIUS = 3.5;
export const TIP_PAD = 4;
export const TIP_GAP = 8;
export const CHART_LINE_WIDTH = 1.5;
export const HAIRLINE_WIDTH = 1;

// DashLen/DashGap as an SVG stroke-dasharray value.
export const DASH_ARRAY = '4 4';

export const PANEL_CORNER_RADIUS = 4;
export const TILE_CORNER_RADIUS = 6;

export const GRID_CELL_WIDTH = 300;
export const CHART_MIN_HEIGHT = 80;

export const FLASH_DURATION_MS = 900;

// Fallback polling cadence when the config carries a non-positive interval
// (mirrors constants.DefaultRefreshInterval).
export const DEFAULT_REFRESH_MS = 30_000;

export const PRICE_PLACEHOLDER = '—';
export const MSG_UNAVAILABLE = 'unavailable';

// Search dialog (mirrors internal/constants text.go / SearchDebounce).
export const SEARCH_DEBOUNCE_MS = 300;
export const PLACEHOLDER_SEARCH = 'Search name or symbol — e.g. Apple or AAPL';
export const MSG_SEARCH_PROMPT = 'Start typing to search…';
export const MSG_SEARCHING = 'Searching…';
export const MSG_SEARCH_FAILED = 'Search failed (offline?)';
export const MSG_NO_MATCHES = 'No matches';
export const TIP_ALREADY_TRACKED = 'Already tracking this index';

// Separators used when joining symbol/name/market fragments.
export const SEP_TITLE = '  ·  ';
export const SEP_META = ' · ';

// Settings view (mirrors internal/constants text.go).
export const TITLE_SETTINGS = 'simpleStonks — Settings';
export const SECTION_GENERAL = 'General';
export const SECTION_APPEARANCE = 'Appearance';
export const SECTION_LOGGING = 'Logging';
export const LABEL_DEFAULT_RANGE = 'Default range';
export const LABEL_REFRESH_SECONDS = 'Refresh interval (seconds)';
export const LABEL_BACKGROUND_COLOR = 'Background color';
export const LABEL_BACKGROUND_OPACITY = 'Background opacity';
export const LABEL_LOG_LEVEL = 'Log level';
export const LABEL_LOG_FILE = 'Log file';
export const LABEL_LOG_MAX_SIZE = 'Max size (MB)';
export const LABEL_LOG_ARCHIVES = 'Archives kept';
export const MSG_ERR_REFRESH_INTERVAL = 'refresh interval must be a whole number of seconds ≥ 1';
export const MSG_ERR_LOG_MAX_SIZE = 'log max size must be a non-negative whole number of MB';
export const MSG_ERR_LOG_ARCHIVES = 'log archives kept must be a non-negative whole number';

// Background defaults (mirrors DefaultBackgroundColor/Opacity in storage constants).
export const DEFAULT_BACKGROUND_COLOR = '#171718';
export const DEFAULT_BACKGROUND_OPACITY = 1;

// EventConfigChanged mirror (constants.EventConfigChanged on the Go side).
export const EVENT_CONFIG_CHANGED = 'configChanged';
