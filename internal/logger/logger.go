package logger

import (
	"context"
	"log/slog"
	"os"

	slogctx "github.com/veqryn/slog-context"
)

const DefaultLogLevel = slog.LevelInfo

type Logger struct {
	*slog.Logger

	lvl *slog.LevelVar
}

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

func (l *Logger) SetLevel(level slog.Level) {
	l.lvl.Set(level)
}

type loggerKey struct{}

func WithLogger(ctx context.Context, logger *Logger) context.Context {
	return context.WithValue(ctx, loggerKey{}, logger)
}

func FromContext(ctx context.Context) *Logger {
	if ctx == nil {
		return New()
	}

	if logger, ok := ctx.Value(loggerKey{}).(*Logger); ok {
		return logger
	}
	return New()
}
