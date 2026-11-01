// Package autocancel runs the order auto-cancellation loop in the worker role.
package autocancel

import (
	"context"
	"log/slog"
	"minimart/internal/order/application"
	"minimart/pkg/background"
	"time"
)

// Runner drives an AutoCanceller through a background runner.
type Runner struct {
	useCase *application.AutoCanceller
	runner  *background.Runner
}

// New builds a runner. A non-positive interval falls back to the background
// runner's default.
func New(useCase *application.AutoCanceller, interval time.Duration, logger *slog.Logger) *Runner {
	if logger == nil {
		logger = slog.Default()
	}
	return &Runner{
		useCase: useCase,
		runner:  background.New(interval, logger, useCase.RunOnce),
	}
}

// NewRunner is an alias for New.
func NewRunner(useCase *application.AutoCanceller, interval time.Duration, logger *slog.Logger) *Runner {
	return New(useCase, interval, logger)
}

// Run blocks until ctx is cancelled.
func (r *Runner) Run(ctx context.Context) {
	if r == nil || r.runner == nil {
		return
	}
	r.runner.Run(ctx)
}
