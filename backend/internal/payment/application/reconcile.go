package application

import (
	"context"
	"errors"
	"log/slog"
	"minimart/internal/payment/domain"
	"time"
)

// Reconciler periodically asks the provider for the status of pending and
// unknown payments. Its RunOnce method is intended to be driven by a
// background runner in the worker role.
type Reconciler struct {
	svc    *Service
	logger *slog.Logger
	age    time.Duration
	limit  int
}

// NewReconciler builds a reconciler over the payment application service. age
// is how stale a payment must be before it is reconciled, and limit is the
// batch size.
func NewReconciler(svc *Service, age time.Duration, limit int, logger *slog.Logger) *Reconciler {
	if logger == nil {
		logger = slog.Default()
	}
	if age <= 0 {
		age = 5 * time.Minute
	}
	if limit <= 0 {
		limit = 100
	}
	return &Reconciler{svc: svc, logger: logger, age: age, limit: limit}
}

// RunOnce reconciles one batch of pending/unknown payments.
func (r *Reconciler) RunOnce(ctx context.Context) {
	payments, err := r.svc.repo.ListReconcilable(ctx, time.Now().Add(-r.age), r.limit)
	if err != nil {
		r.logger.WarnContext(ctx, "payment reconciliation list failed", "error", err)
		return
	}
	for _, p := range payments {
		if p.ProviderRef == nil || *p.ProviderRef == "" {
			continue
		}
		res, err := r.svc.provider.GetStatus(ctx, *p.ProviderRef)
		if err != nil {
			r.logger.WarnContext(ctx, "payment reconciliation status lookup failed", "payment_id", p.ID, "error", err)
			continue
		}
		// A charge the provider confirms as captured on a cancelled order must
		// be reversed, not finalized: refund it instead of reconciling it, so
		// the succeeded state it skips never becomes visible.
		if res.Outcome == domain.OutcomeApproved && r.orderCancelled(ctx, p.OrderID) {
			if _, refundErr := r.svc.refund(ctx, p, refundReasonOrderCancelled); refundErr != nil {
				if !errors.Is(refundErr, domain.ErrPaymentNotRefundable) {
					r.logger.WarnContext(ctx, "payment reconciliation refund failed", "payment_id", p.ID, "error", refundErr)
				}
			}
			continue
		}
		updated, err := p.ApplyReconciliation(res)
		if err != nil {
			if !errors.Is(err, domain.ErrPaymentNotPending) {
				r.logger.WarnContext(ctx, "payment reconciliation apply failed", "payment_id", p.ID, "error", err)
			}
			continue
		}
		if updated.Status == p.Status {
			continue
		}
		if err := r.svc.repo.Reconcile(ctx, updated); err != nil {
			if !errors.Is(err, domain.ErrPaymentNotPending) {
				r.logger.WarnContext(ctx, "payment reconciliation persist failed", "payment_id", p.ID, "error", err)
			}
			continue
		}
		r.svc.m.IncPayment(string(updated.Status))
	}
}

// orderCancelled reports whether the order of a payment being reconciled has
// been cancelled. It fails open: when the answer cannot be determined the
// payment is reconciled normally, and the refund is left to a later retry or
// the manual refund endpoint.
func (r *Reconciler) orderCancelled(ctx context.Context, orderID string) bool {
	cancelled, err := r.svc.orderCancelled(ctx, orderID)
	if err != nil {
		r.logger.WarnContext(ctx, "payment reconciliation order lookup failed", "order_id", orderID, "error", err)
		return false
	}
	return cancelled
}
