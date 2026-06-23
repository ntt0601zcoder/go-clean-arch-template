// Package logger is the single logging substitute for the reference's
// interface-common logger/zerolog. It builds component-scoped *slog.Logger
// instances and carries a logger through context.Context.
package logger

import (
	"context"
	"log/slog"
	"os"
	"strings"
)

type ctxKey struct{}

// NewSlogger returns a JSON slog.Logger tagged with the component name. level is
// "debug|info|warn|error" (default info).
func NewSlogger(component, level string) *slog.Logger {
	h := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: parseLevel(level)})
	return slog.New(h).With(slog.String("component", component))
}

func parseLevel(level string) slog.Level {
	switch strings.ToLower(level) {
	case "debug":
		return slog.LevelDebug
	case "warn", "warning":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}

// Into returns a copy of ctx carrying log.
func Into(ctx context.Context, log *slog.Logger) context.Context {
	return context.WithValue(ctx, ctxKey{}, log)
}

// From extracts the logger stored by Into, falling back to slog.Default.
func From(ctx context.Context) *slog.Logger {
	if log, ok := ctx.Value(ctxKey{}).(*slog.Logger); ok && log != nil {
		return log
	}
	return slog.Default()
}
