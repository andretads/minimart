// Package reconciliation runs the payment reconciliation loop in the worker role.
package reconciliation

import (
	"context"
	"log/slog"
	"minimart/internal/payment/application"
	"minimart/pkg/background"
	"time"
)

// Runner drives a Reconciler through a background runner.
type Runner struct {
	useCase *application.Reconciler
	runner  *background.Runner
}

// New builds a runner. A non-positive interval falls back to the background
// runner's default.
func New(useCase *application.Reconciler, interval time.Duration, logger *slog.Logger) *Runner {
	if logger == nil {
		logger = slog.Default()
	}
	return &Runner{
		useCase: useCase,
		runner:  background.New(interval, logger, useCase.RunOnce),
	}
}

// NewRunner is an alias for New.
func NewRunner(useCase *application.Reconciler, interval time.Duration, logger *slog.Logger) *Runner {
	return New(useCase, interval, logger)
}

// Run blocks until ctx is cancelled.
func (r *Runner) Run(ctx context.Context) {
	if r == nil || r.runner == nil {
		return
	}
	r.runner.Run(ctx)
}
