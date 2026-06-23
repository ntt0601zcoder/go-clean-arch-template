// Package gorm provides a GORM logger that bridges to slog. Import it aliased
// (e.g. cgorm) to avoid colliding with gorm.io/gorm.
package gorm

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"gorm.io/gorm/logger"
)

// slogLogger adapts *slog.Logger to gorm's logger.Interface.
type slogLogger struct {
	log       *slog.Logger
	level     logger.LogLevel
	slowQuery time.Duration
}

// NewGormLogger builds a gorm logger at the given level ("silent|error|warn|info").
func NewGormLogger(log *slog.Logger, level string) logger.Interface {
	return &slogLogger{log: log.With(slog.String("source", "gorm")), level: parseLevel(level), slowQuery: 200 * time.Millisecond}
}

func parseLevel(level string) logger.LogLevel {
	switch level {
	case "silent":
		return logger.Silent
	case "error":
		return logger.Error
	case "warn", "warning":
		return logger.Warn
	default:
		return logger.Info
	}
}

func (l *slogLogger) LogMode(level logger.LogLevel) logger.Interface {
	cp := *l
	cp.level = level
	return &cp
}

func (l *slogLogger) Info(ctx context.Context, msg string, data ...any) {
	if l.level >= logger.Info {
		l.log.InfoContext(ctx, msg, slog.Any("data", data))
	}
}

func (l *slogLogger) Warn(ctx context.Context, msg string, data ...any) {
	if l.level >= logger.Warn {
		l.log.WarnContext(ctx, msg, slog.Any("data", data))
	}
}

func (l *slogLogger) Error(ctx context.Context, msg string, data ...any) {
	if l.level >= logger.Error {
		l.log.ErrorContext(ctx, msg, slog.Any("data", data))
	}
}

func (l *slogLogger) Trace(ctx context.Context, begin time.Time, fc func() (string, int64), err error) {
	if l.level <= logger.Silent {
		return
	}
	elapsed := time.Since(begin)
	sql, rows := fc()
	attrs := []any{slog.String("sql", sql), slog.Int64("rows", rows), slog.Duration("elapsed", elapsed)}

	switch {
	case err != nil && l.level >= logger.Error && !errors.Is(err, logger.ErrRecordNotFound):
		l.log.ErrorContext(ctx, "gorm query", append(attrs, slog.Any("error", err))...)
	case elapsed > l.slowQuery && l.level >= logger.Warn:
		l.log.WarnContext(ctx, "gorm slow query", attrs...)
	case l.level >= logger.Info:
		l.log.DebugContext(ctx, "gorm query", attrs...)
	}
}
