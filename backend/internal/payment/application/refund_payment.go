package application

import (
	"context"
	"fmt"
	"minimart/internal/payment/domain"
)

// Refund reasons ride the PaymentRefunded event, so consumers can explain a
// refund without a lookup.
const (
	// refundReasonOrderCancelled explains automatic refunds triggered by an
	// order cancellation.
	refundReasonOrderCancelled = "pedido cancelado"
	// refundReasonManual explains operator-initiated refunds.
	refundReasonManual = "reembolso solicitado"
)

// RefundPayment refunds a captured payment by ID, idempotently: refunding an
// already-refunded payment returns it unchanged. Only a succeeded payment for
// a cancelled order can be refunded this way; reversing a payment on a live
// order is a return, which is out of scope.
func (s *Service) RefundPayment(ctx context.Context, id string) (domain.Payment, error) {
	p, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return domain.Payment{}, err
	}
	if p.Status == domain.StatusRefunded {
		return p, nil
	}
	if p.Status != domain.StatusSucceeded {
		return domain.Payment{}, domain.ErrPaymentNotRefundable
	}
	cancelled, err := s.orderCancelled(ctx, p.OrderID)
	if err != nil {
		return domain.Payment{}, err
	}
	if !cancelled {
		return domain.Payment{}, ErrOrderNotCancelled
	}
	return s.refund(ctx, p, refundReasonManual)
}

// refund reverses p's charge with the provider and persists the refunded
// state, raising PaymentRefunded atomically with it. p must be a captured
// payment carrying the provider reference of the charge to reverse: pending
// (captured but not yet finalized), succeeded, or unknown-but-confirmed
// (reconciliation proved the charge landed).
func (s *Service) refund(ctx context.Context, p domain.Payment, reason string) (domain.Payment, error) {
	if p.Status != domain.StatusPending && p.Status != domain.StatusSucceeded && p.Status != domain.StatusUnknown {
		return domain.Payment{}, domain.ErrPaymentNotRefundable
	}
	if p.ProviderRef == nil || *p.ProviderRef == "" {
		return domain.Payment{}, domain.ErrPaymentNotRefundable
	}
	res, err := s.provider.Refund(ctx, *p.ProviderRef, p.AmountCents)
	if err != nil {
		return domain.Payment{}, fmt.Errorf("%w: %w", ErrRefundFailed, err)
	}
	if res.Outcome != domain.OutcomeApproved {
		return domain.Payment{}, fmt.Errorf("%w: %s", ErrRefundFailed, res.Reason)
	}
	res.Reason = reason
	refunded, err := p.Refund(res)
	if err != nil {
		return domain.Payment{}, err
	}
	if err := s.repo.Refund(ctx, refunded); err != nil {
		return domain.Payment{}, err
	}
	s.m.IncPayment(string(refunded.Status))
	return refunded, nil
}

// orderCancelled reports whether the order has been cancelled. A reader error
// is returned to the caller, which decides whether to fail open (the charge
// proceeds and the reconciler settles later) or closed (the refund is
// refused).
func (s *Service) orderCancelled(ctx context.Context, orderID string) (bool, error) {
	if s.orderReader == nil {
		return false, nil
	}
	ord, err := s.orderReader.GetOrder(ctx, orderID)
	if err != nil {
		return false, err
	}
	return ord.Status == "cancelled", nil
}
