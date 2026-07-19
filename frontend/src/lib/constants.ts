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

// EventConfigChanged mirror (constants.EventConfigChanged on the Go side).
export const EVENT_CONFIG_CHANGED = 'configChanged';
