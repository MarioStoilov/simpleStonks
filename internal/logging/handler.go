package logging

import (
	"context"
	"log/slog"
	"sync"
)

// switchHandler is an slog.Handler that delegates to an inner handler which can
// be swapped atomically. This keeps the top-level *slog.Logger stable while the
// underlying destination and level change on reconfigure, so a reference held
// elsewhere (or installed as slog's default) keeps working across changes.
type switchHandler struct {
	mutex sync.RWMutex
	inner slog.Handler
}

func (handler *switchHandler) set(next slog.Handler) {
	handler.mutex.Lock()
	handler.inner = next
	handler.mutex.Unlock()
}

func (handler *switchHandler) current() slog.Handler {
	handler.mutex.RLock()
	defer handler.mutex.RUnlock()
	return handler.inner
}

func (handler *switchHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return handler.current().Enabled(ctx, level)
}

func (handler *switchHandler) Handle(ctx context.Context, record slog.Record) error {
	return handler.current().Handle(ctx, record)
}

// WithAttrs and WithGroup delegate to the current inner handler. Loggers derived
// through these bind to the destination in effect at derivation time and do not
// hot-swap; the root logger (used as the default) does.
func (handler *switchHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return handler.current().WithAttrs(attrs)
}

func (handler *switchHandler) WithGroup(name string) slog.Handler {
	return handler.current().WithGroup(name)
}

// discardHandler drops every record; used for the silent level.
type discardHandler struct{}

func (discardHandler) Enabled(context.Context, slog.Level) bool  { return false }
func (discardHandler) Handle(context.Context, slog.Record) error { return nil }
func (discardHandler) WithAttrs([]slog.Attr) slog.Handler        { return discardHandler{} }
func (discardHandler) WithGroup(string) slog.Handler             { return discardHandler{} }
