package logging

import (
	"context"
	"log/slog"
)

var NopSlogger *slog.Logger

func init() {
	NopSlogger = slog.New(&discardHandler{})
}

// taken from here: https://go-review.googlesource.com/c/go/+/548335/5/src/log/slog/example_discard_test.go
type discardHandler struct {
	slog.JSONHandler
}

func (d *discardHandler) Enabled(context.Context, slog.Level) bool { return false }
