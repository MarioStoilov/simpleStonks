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

	"github.com/MarioStoilov/simplestonks/internal/model"
)

// appDir is the config subdirectory and file names used under UserConfigDir.
const (
	appDir     = "simplestonks"
	configFile = "config.json"
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
}

// Default returns the configuration used on first run.
func Default() Config {
	return Config{
		Symbols:         []string{"^GSPC", "^IXIC", "AAPL"},
		DefaultRange:    model.Range1D,
		Layout:          LayoutGrid,
		FormFactor:      FormFactorWindow,
		RefreshInterval: 30 * time.Second,
	}
}

// Path returns the absolute path to the config file.
func Path() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, appDir, configFile), nil
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
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}
