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
	"github.com/MarioStoilov/simplestonks/internal/constants"
)

// Logger is a leveled logger whose destination and level can be reconfigured at
// runtime (e.g. on a config live-reload) without invalidating held references.
type Logger struct {
	handler *switchHandler
	slogger *slog.Logger

	mutex  sync.Mutex
	writer io.Closer // current rotating writer, if any
}

// New builds a Logger for the given configuration.
func New(cfg config.Logging) (*Logger, error) {
	logger := &Logger{handler: &switchHandler{inner: discardHandler{}}}
	logger.slogger = slog.New(logger.handler)
	if err := logger.Reconfigure(cfg); err != nil {
		return nil, err
	}
	return logger, nil
}

// Slog returns the stable *slog.Logger. It keeps working across Reconfigure.
func (logger *Logger) Slog() *slog.Logger { return logger.slogger }

// Reconfigure swaps the logger's level and destination to match cfg, closing the
// previous writer. It is safe to call concurrently with logging on l.Slog().
func (logger *Logger) Reconfigure(cfg config.Logging) error {
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
		rotating, err := newRotatingWriter(path, int64(cfg.MaxSizeMB)*constants.BytesPerMiB, cfg.MaxArchives)
		if err != nil {
			return err
		}
		newHandler = slog.NewTextHandler(rotating, &slog.HandlerOptions{Level: level})
		newWriter = rotating
	}

	logger.handler.set(newHandler)

	logger.mutex.Lock()
	old := logger.writer
	logger.writer = newWriter
	logger.mutex.Unlock()
	if old != nil {
		_ = old.Close()
	}
	return nil
}

// Close releases the current writer.
func (logger *Logger) Close() error {
	logger.mutex.Lock()
	defer logger.mutex.Unlock()
	if logger.writer == nil {
		return nil
	}
	err := logger.writer.Close()
	logger.writer = nil
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
