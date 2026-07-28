// Package config loads and persists simpleStonks' user configuration: the list
// of tracked symbols plus application settings.
//
// The config file is a JSON document stored under os.UserConfigDir(), which
// resolves to $XDG_CONFIG_HOME (and to $SNAP_USER_DATA/.config under snap
// confinement), so the same code path works for both native and snap builds.
package config

import (
	"encoding/json"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"time"

	"github.com/MarioStoilov/simplestonks/internal/constants"
	"github.com/MarioStoilov/simplestonks/internal/model"
)

// Layout selects how tracked symbols are arranged. v1 ships Grid; ListDetail is
// architected-for and added later.
type Layout string

const (
	LayoutGrid       Layout = "grid"
	LayoutListDetail Layout = "list-detail"
)

// FormFactor selects the window style. v1 ships Window; Widget (always-on-top)
// is architected-for and added later.
type FormFactor string

const (
	FormFactorWindow FormFactor = "window"
	FormFactorWidget FormFactor = "widget"
)

// LogLevel controls logging verbosity, ordered from silent (no output at all)
// to debug (most verbose).
type LogLevel string

const (
	LogSilent LogLevel = "silent"
	LogError  LogLevel = "error"
	LogWarn   LogLevel = "warn"
	LogInfo   LogLevel = "info"
	LogDebug  LogLevel = "debug"
)

// Logging is the logger configuration. It is part of Config, so it is persisted
// and live-reloaded like every other setting.
type Logging struct {
	// Level is the verbosity, from LogSilent to LogDebug.
	Level LogLevel `json:"level"`

	// File is the log file path. Empty means DefaultLogPath().
	File string `json:"file"`

	// MaxSizeMB is the size threshold (in MB) at which the log file is rotated.
	// Zero or negative disables rotation.
	MaxSizeMB int `json:"maxSizeMB"`

	// MaxArchives is how many rotated archive files to retain. Zero keeps none
	// (the log simply starts fresh when the threshold is hit).
	MaxArchives int `json:"maxArchives"`
}

// DefaultLogPath returns the default log file location, following the XDG state
// directory convention. Under snap confinement $HOME points at the snap's data
// dir, so this stays writable there too.
func DefaultLogPath() string {
	base := os.Getenv(constants.EnvXDGStateHome)
	if base == "" {
		if home, err := os.UserHomeDir(); err == nil {
			base = filepath.Join(home, constants.StateSubdir)
		} else if cache, err := os.UserCacheDir(); err == nil {
			base = cache
		}
	}
	return filepath.Join(base, constants.AppDirName, constants.LogFileName)
}

// Background styles the app window background.
type Background struct {
	// Color is the background color as a "#RRGGBB" hex string.
	Color string `json:"color"`

	// Opacity is the background opacity from 0 (fully transparent) to 1
	// (fully opaque).
	Opacity float64 `json:"opacity"`
}

// DefaultBackground matches Fyne's dark-theme window background.
func DefaultBackground() Background {
	return Background{
		Color:   constants.DefaultBackgroundColor,
		Opacity: constants.DefaultBackgroundOpacity,
	}
}

// Chart styles the chart plot area.
type Chart struct {
	// Background is the plot background color as a "#RRGGBB" hex string.
	Background string `json:"background"`

	// Grid draws a graph-paper grid over the plot background.
	Grid bool `json:"grid"`

	// GridSize is the grid square size in pixels.
	GridSize int `json:"gridSize"`

	// GridColor is the grid line color as a "#RRGGBB" hex string.
	GridColor string `json:"gridColor"`

	// Fill shades the area between the price line and the previous-close
	// reference — green above, red below (the logo look).
	Fill bool `json:"fill"`

	// FillOpacity is the fill opacity from 0 (invisible) to 1 (solid).
	FillOpacity float64 `json:"fillOpacity"`
}

// DefaultChart is the out-of-the-box chart styling: the logo-style area fill
// on, the grid off but ready with subtle defaults.
func DefaultChart() Chart {
	return Chart{
		Background:  constants.DefaultChartBackground,
		Grid:        false,
		GridSize:    constants.DefaultChartGridSize,
		GridColor:   constants.DefaultChartGridColor,
		Fill:        true,
		FillOpacity: constants.DefaultChartFillOpacity,
	}
}

// Config is the full persisted configuration.
type Config struct {
	// Symbols is the ordered list of tracked tickers/indexes (e.g. "AAPL", "^GSPC").
	Symbols []string `json:"symbols"`

	// DefaultRange is the range the app opens with.
	DefaultRange model.Range `json:"defaultRange"`

	// Layout and FormFactor control presentation.
	Layout     Layout     `json:"layout"`
	FormFactor FormFactor `json:"formFactor"`

	// RefreshInterval controls live-tick polling cadence for the 1D view.
	RefreshInterval time.Duration `json:"refreshInterval"`

	// ExtendedHours shows pre-market/after-hours data on the detail view's
	// 1D chart while the market is outside regular hours.
	ExtendedHours bool `json:"extendedHours"`

	// Background styles the app window background.
	Background Background `json:"background"`

	// Chart styles the chart plot area (background, grid, area fill).
	Chart Chart `json:"chart"`

	// Logging configures the leveled, rotating file logger.
	Logging Logging `json:"logging"`
}

// clone returns a copy safe to mutate without aliasing the original's slices.
// Update relies on this so in-place edits (e.g. reordering Symbols) don't also
// mutate the stored config and defeat its change detection.
func (cfg Config) clone() Config {
	copied := cfg
	copied.Symbols = append([]string(nil), cfg.Symbols...)
	return copied
}

// Default returns the configuration used on first run.
func Default() Config {
	return Config{
		Symbols:         append([]string(nil), constants.DefaultSymbols...),
		DefaultRange:    model.Range1D,
		Layout:          LayoutGrid,
		FormFactor:      FormFactorWindow,
		RefreshInterval: constants.DefaultRefreshInterval,
		ExtendedHours:   true,
		Background:      DefaultBackground(),
		Chart:           DefaultChart(),
		Logging: Logging{
			Level: LogInfo,
			// File stays empty so the default path is resolved at runtime:
			// an absolute path baked in here would go stale — under snap it
			// contains the revision directory, which changes on refresh.
			File:        "",
			MaxSizeMB:   constants.DefaultLogMaxSizeMB,
			MaxArchives: constants.DefaultLogMaxArchives,
		},
	}
}

// Path returns the absolute path to the config file.
func Path() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, constants.AppDirName, constants.ConfigFileName), nil
}

// Load reads the config file, returning Default() (and no error) when the file
// does not yet exist so first run just works.
func Load() (Config, error) {
	path, err := Path()
	if err != nil {
		return Config{}, err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return Default(), nil
		}
		return Config{}, err
	}
	cfg := Default()
	if err := json.Unmarshal(data, &cfg); err != nil {
		return Config{}, err
	}
	return cfg, nil
}

// Save writes the config to disk, creating the config directory if needed. The
// write is atomic: it writes to a temp file and renames over the target.
func Save(cfg Config) error {
	path, err := Path()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), constants.DirPerm); err != nil {
		return err
	}
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	tmp := path + constants.TmpFileSuffix
	if err := os.WriteFile(tmp, data, constants.FilePerm); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}
