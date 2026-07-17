// Package logging provides simpleStonks' leveled, rotating file logger.
//
// Verbosity runs from silent (no output) to debug, is stored in the config file,
// and is applied live when the config is reloaded. The logger writes to a
// configurable file that rotates once it passes a size threshold, keeping a
// bounded number of archives. The top-level *slog.Logger is stable across
// reconfiguration so it can be installed as slog's default.
package logging

import (
	"io"
	"log/slog"
	"sync"

	"github.com/MarioStoilov/simplestonks/internal/config"
)

const mib = 1024 * 1024

// Logger is a leveled logger whose destination and level can be reconfigured at
// runtime (e.g. on a config live-reload) without invalidating held references.
type Logger struct {
	handler *switchHandler
	sl      *slog.Logger

	mu     sync.Mutex
	writer io.Closer // current rotating writer, if any
}

// New builds a Logger for the given configuration.
func New(cfg config.Logging) (*Logger, error) {
	l := &Logger{handler: &switchHandler{inner: discardHandler{}}}
	l.sl = slog.New(l.handler)
	if err := l.Reconfigure(cfg); err != nil {
		return nil, err
	}
	return l, nil
}

// Slog returns the stable *slog.Logger. It keeps working across Reconfigure.
func (l *Logger) Slog() *slog.Logger { return l.sl }

// Reconfigure swaps the logger's level and destination to match cfg, closing the
// previous writer. It is safe to call concurrently with logging on l.Slog().
func (l *Logger) Reconfigure(cfg config.Logging) error {
	level, silent := slogLevel(cfg.Level)

	var (
		newHandler slog.Handler
		newWriter  io.Closer
	)
	if silent {
		newHandler = discardHandler{}
	} else {
		path := cfg.File
		if path == "" {
			path = config.DefaultLogPath()
		}
		rw, err := newRotatingWriter(path, int64(cfg.MaxSizeMB)*mib, cfg.MaxArchives)
		if err != nil {
			return err
		}
		newHandler = slog.NewTextHandler(rw, &slog.HandlerOptions{Level: level})
		newWriter = rw
	}

	l.handler.set(newHandler)

	l.mu.Lock()
	old := l.writer
	l.writer = newWriter
	l.mu.Unlock()
	if old != nil {
		_ = old.Close()
	}
	return nil
}

// Close releases the current writer.
func (l *Logger) Close() error {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.writer == nil {
		return nil
	}
	err := l.writer.Close()
	l.writer = nil
	return err
}

// slogLevel maps a config.LogLevel to an slog.Level, reporting whether logging
// is silenced entirely.
func slogLevel(lvl config.LogLevel) (level slog.Level, silent bool) {
	switch lvl {
	case config.LogSilent:
		return 0, true
	case config.LogError:
		return slog.LevelError, false
	case config.LogWarn:
		return slog.LevelWarn, false
	case config.LogDebug:
		return slog.LevelDebug, false
	default: // LogInfo and any unknown value
		return slog.LevelInfo, false
	}
}
