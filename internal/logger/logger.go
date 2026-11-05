// Package logger provides a contextual slog logger with an adjustable level.
package logger

import (
	"context"
	"log/slog"
	"os"

	slogctx "github.com/veqryn/slog-context"
)

// DefaultLogLevel is the level new loggers log at.
const DefaultLogLevel = slog.LevelInfo

// Logger is a slog.Logger whose level can be changed.
type Logger struct {
	*slog.Logger

	lvl *slog.LevelVar
}

// New returns a Logger that writes text to stdout at DefaultLogLevel.
func New() *Logger {
	// Set default log level
	lvl := new(slog.LevelVar)
	lvl.Set(DefaultLogLevel)

	// Basic slog handler
	slogHandler := slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: lvl,
	})

	// Contextual slog handler
	slogctxHandler := slogctx.NewHandler(slogHandler, nil)
	logger := slog.New(slogctxHandler)

	return &Logger{
		Logger: logger,
		lvl:    lvl,
	}
}

// SetLevel sets the minimum level the logger logs at.
func (l *Logger) SetLevel(level slog.Level) {
	l.lvl.Set(level)
}

type loggerKey struct{}

// WithLogger returns a copy of ctx that carries logger.
func WithLogger(ctx context.Context, logger *Logger) context.Context {
	return context.WithValue(ctx, loggerKey{}, logger)
}

// FromContext returns the logger carried by ctx, or a new one.
func FromContext(ctx context.Context) *Logger {
	if ctx == nil {
		return New()
	}

	if logger, ok := ctx.Value(loggerKey{}).(*Logger); ok {
		return logger
	}
	return New()
}
