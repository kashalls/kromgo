// Package logging centralizes kromgo's slog setup and carries a request-scoped
// logger through context so middleware and handlers share one set of base fields.
package logging

import (
	"context"
	"io"
	"log/slog"
	"os"
	"strings"
)

// Init configures the default slog logger from LOG_LEVEL (debug/info/warn/error,
// default info) and LOG_FORMAT (text or json, default json).
func Init() {
	slog.SetDefault(slog.New(newHandler(os.Stdout, os.Getenv("LOG_LEVEL"), os.Getenv("LOG_FORMAT"))))
}

// newHandler builds a JSON (default) or text slog handler at the given level.
func newHandler(w io.Writer, level, format string) slog.Handler {
	opts := &slog.HandlerOptions{Level: parseLevel(level)}
	if strings.EqualFold(format, "text") {
		return slog.NewTextHandler(w, opts)
	}
	return slog.NewJSONHandler(w, opts)
}

func parseLevel(s string) slog.Level {
	switch strings.ToLower(s) {
	case "debug":
		return slog.LevelDebug
	case "warn":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}

type ctxKey struct{}

// WithLogger returns a copy of ctx carrying l, retrievable via FromContext.
func WithLogger(ctx context.Context, l *slog.Logger) context.Context {
	return context.WithValue(ctx, ctxKey{}, l)
}

// FromContext returns the logger stored in ctx, or slog.Default() if none is set
// (e.g. handlers exercised in tests without the server middleware).
func FromContext(ctx context.Context) *slog.Logger {
	if l, ok := ctx.Value(ctxKey{}).(*slog.Logger); ok {
		return l
	}
	return slog.Default()
}
