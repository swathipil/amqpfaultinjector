package logging

import (
	"context"
	"log/slog"
)

type sloggerKey struct{}

var _key = sloggerKey{}

// ContextWithSlogger adds a slogger into the context.
func ContextWithSlogger(ctx context.Context, logger *slog.Logger) context.Context {
	return context.WithValue(ctx, _key, logger)
}

// SloggerFromContext gets the slogger in the context, or the default slogger.
func SloggerFromContext(ctx context.Context) *slog.Logger {
	logger, ok := ctx.Value(_key).(*slog.Logger)

	if !ok {
		return slog.Default()
	}

	return logger
}

// ContextWithSloggerAndValues extracts the slogger from the context, creates a new child slogger
// with the new values, and returns a context with the new slogger.
func ContextWithSloggerAndValues(ctx context.Context, values ...any) (context.Context, *slog.Logger) {
	logger := SloggerFromContext(ctx)
	newLogger := logger.With(values...)
	return ContextWithSlogger(ctx, newLogger), newLogger
}
