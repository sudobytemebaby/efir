package logger

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"strings"
)

type Level = slog.Level

const (
	LevelDebug = slog.LevelDebug
	LevelInfo  = slog.LevelInfo
	LevelWarn  = slog.LevelWarn
	LevelError = slog.LevelError
)

type Options struct {
	Level  Level
	Output io.Writer
}

func New(opts Options) *slog.Logger {
	if opts.Output == nil {
		opts.Output = os.Stdout
	}

	return slog.New(slog.NewJSONHandler(opts.Output, &slog.HandlerOptions{
		Level: opts.Level,
	}))
}

// ParseLevel converts a string log level to slog.Level.
// Valid values: debug, info, warn, error (case-insensitive).
// Returns LevelInfo and an error if the value is unrecognized.
func ParseLevel(s string) (Level, error) {
	switch strings.ToLower(s) {
	case "debug":
		return LevelDebug, nil
	case "info":
		return LevelInfo, nil
	case "warn":
		return LevelWarn, nil
	case "error":
		return LevelError, nil
	default:
		return LevelInfo, fmt.Errorf("unknown log level %q, using info", s)
	}
}

func WithContext(ctx context.Context, logger *slog.Logger) context.Context {
	return context.WithValue(ctx, contextKey{}, logger)
}

func FromContext(ctx context.Context) *slog.Logger {
	if logger, ok := ctx.Value(contextKey{}).(*slog.Logger); ok {
		return logger
	}
	return slog.Default()
}

type contextKey struct{}
