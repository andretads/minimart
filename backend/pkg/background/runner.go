// Package background provides a generic, context-aware ticker runner for use
// cases that need to run periodically in the worker role.
package background

import (
	"context"
	"log/slog"
	"time"
)

// Runner periodically invokes a function until its context is cancelled.
type Runner struct {
	interval time.Duration
	fn       func(context.Context)
	logger   *slog.Logger
}

// New builds a runner. A non-positive interval falls back to one second.
func New(interval time.Duration, logger *slog.Logger, fn func(context.Context)) *Runner {
	if logger == nil {
		logger = slog.Default()
	}
	if interval <= 0 {
		interval = time.Second
	}
	return &Runner{interval: interval, logger: logger, fn: fn}
}

// NewRunner is an alias for New.
func NewRunner(interval time.Duration, logger *slog.Logger, fn func(context.Context)) *Runner {
	return New(interval, logger, fn)
}

// NewTicker is an alias for New.
func NewTicker(interval time.Duration, logger *slog.Logger, fn func(context.Context)) *Runner {
	return New(interval, logger, fn)
}

// Run blocks until ctx is cancelled, invoking fn immediately and on every tick.
func (r *Runner) Run(ctx context.Context) {
	if r.fn == nil {
		return
	}
	r.fn(ctx)
	ticker := time.NewTicker(r.interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			r.fn(ctx)
		}
	}
}
