package utils

import (
	"context"
	"errors"
)

// RunGoroutine runs a goroutine that can be cancelled using ctx.
// It returns a channel that is closed when the goroutine exits, and will contain an error
// if one occurred that was not a cancellation error (context.DeadlineExceeded or context.Canceled).
func RunGoroutine(ctx context.Context, fn func(ctx context.Context) error) (done chan error) {
	done = make(chan error, 1)

	go func() {
		defer close(done)

		err := fn(ctx)

		switch {
		case err == nil:
		case errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded):
		default:
			done <- err
		}
	}()

	return done
}
