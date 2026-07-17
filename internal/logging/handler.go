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
	mu    sync.RWMutex
	inner slog.Handler
}

func (s *switchHandler) set(h slog.Handler) {
	s.mu.Lock()
	s.inner = h
	s.mu.Unlock()
}

func (s *switchHandler) current() slog.Handler {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.inner
}

func (s *switchHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return s.current().Enabled(ctx, level)
}

func (s *switchHandler) Handle(ctx context.Context, r slog.Record) error {
	return s.current().Handle(ctx, r)
}

// WithAttrs and WithGroup delegate to the current inner handler. Loggers derived
// through these bind to the destination in effect at derivation time and do not
// hot-swap; the root logger (used as the default) does.
func (s *switchHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return s.current().WithAttrs(attrs)
}

func (s *switchHandler) WithGroup(name string) slog.Handler {
	return s.current().WithGroup(name)
}

// discardHandler drops every record; used for the silent level.
type discardHandler struct{}

func (discardHandler) Enabled(context.Context, slog.Level) bool  { return false }
func (discardHandler) Handle(context.Context, slog.Record) error { return nil }
func (discardHandler) WithAttrs([]slog.Attr) slog.Handler        { return discardHandler{} }
func (discardHandler) WithGroup(string) slog.Handler             { return discardHandler{} }
