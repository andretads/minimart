package application

import (
	"context"
	"errors"
	"log/slog"
	"minimart/internal/order/domain"
	"minimart/pkg/metrics"
	"time"
)

// AutoCanceller cancels placed orders that have remained unpaid past
// domain.AutoCancelAfter. Its RunOnce method is intended to be driven by a
// background runner in the worker role.
type AutoCanceller struct {
	repo   domain.Repository
	m      metrics.Metrics
	logger *slog.Logger
	limit  int
}

// NewAutoCanceller builds an auto-canceller over the order repository.
func NewAutoCanceller(repo domain.Repository, m metrics.Metrics, limit int, logger *slog.Logger) *AutoCanceller {
	if logger == nil {
		logger = slog.Default()
	}
	if limit <= 0 {
		limit = 100
	}
	return &AutoCanceller{repo: repo, m: m, logger: logger, limit: limit}
}

// RunOnce cancels one batch of stale placed orders.
func (a *AutoCanceller) RunOnce(ctx context.Context) {
	orders, err := a.repo.ListPlacedBefore(ctx, time.Now().UTC().Add(-domain.AutoCancelAfter), a.limit)
	if err != nil {
		a.logger.WarnContext(ctx, "order auto-cancel list failed", "error", err)
		return
	}
	for _, o := range orders {
		cancelled, err := o.Cancel()
		if err != nil {
			if !errors.Is(err, domain.ErrOrderAlreadyCancelled) && !errors.Is(err, domain.ErrOrderNotCancellable) {
				a.logger.WarnContext(ctx, "order auto-cancel failed", "order_id", o.ID, "error", err, "reason", "auto_cancel")
			}
			continue
		}
		if err := a.repo.Update(ctx, cancelled, domain.StatusPlaced); err != nil {
			if errors.Is(err, domain.ErrOrderStateChanged) || errors.Is(err, domain.ErrOrderNotFound) {
				continue
			}
			a.logger.WarnContext(ctx, "order auto-cancel persist failed", "order_id", o.ID, "error", err, "reason", "auto_cancel")
			continue
		}
		a.m.IncOrderCancelled(metrics.StatusSuccess)
		a.logger.InfoContext(ctx, "order auto-cancelled", "order_id", o.ID, "reason", "auto_cancel")
	}
}
