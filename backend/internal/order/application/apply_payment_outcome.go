package application

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"minimart/internal/order/domain"
	"minimart/pkg/metrics"
)

// PaymentOutcome is a terminal payment result. The payment domain publishes its
// outcomes as events; the order consumer maps them onto this enum so the order
// use case never imports the payment module.
type PaymentOutcome string

const (
	// PaymentOutcomeSucceeded is a captured payment: the order becomes paid.
	PaymentOutcomeSucceeded PaymentOutcome = "succeeded"
	// PaymentOutcomeFailed is a declined payment: the order becomes payment_failed.
	PaymentOutcomeFailed PaymentOutcome = "failed"
	// PaymentOutcomeUnknown is a provider timeout or technical error: the order
	// becomes payment_unknown until the payment is reconciled.
	PaymentOutcomeUnknown PaymentOutcome = "unknown"
)

// ApplyPaymentOutcome brings an order's status in line with a terminal payment
// outcome and persists it as a compare-and-set against the status it was read
// with, so a racing transition is never clobbered.
//
// It is idempotent: an outcome that does not move the order (a redelivered event,
// an order already paid or cancelled, a settled failure) writes nothing and
// returns nil, which is what lets an at-least-once consumer ack safely. Losing a
// status race (domain.ErrOrderStateChanged) is likewise a benign no-op: another
// outcome already landed and is authoritative.
func (s *Service) ApplyPaymentOutcome(ctx context.Context, orderID string, outcome PaymentOutcome, reason string) error {
	if err := s.applyPaymentOutcome(ctx, orderID, outcome, reason); err != nil {
		s.m.IncOrderPaymentApplied(metrics.StatusError)
		return err
	}
	s.m.IncOrderPaymentApplied(metrics.StatusSuccess)
	return nil
}

func (s *Service) applyPaymentOutcome(ctx context.Context, orderID string, outcome PaymentOutcome, reason string) error {
	order, err := s.repo.GetByID(ctx, orderID)
	if err != nil {
		return err
	}
	prev := order.Status

	var updated domain.Order
	switch outcome {
	case PaymentOutcomeSucceeded:
		// An order placed with shipping carries the snapshot the shipping domain
		// needs to create the shipment, so paying it raises the order-paid event.
		// An order placed without shipping has nothing to hand over.
		if order.ShippingQuoteID != "" && order.ShippingMethodID != "" {
			updated = order.MarkPaidWithShipping()
		} else {
			updated = order.MarkPaid()
		}
	case PaymentOutcomeFailed:
		updated = order.MarkPaymentFailed()
	case PaymentOutcomeUnknown:
		updated = order.MarkPaymentUnknown()
	default:
		return fmt.Errorf("unknown payment outcome %q", outcome)
	}

	if updated.Status == prev {
		// Nothing this outcome may change: either the event was redelivered or the
		// order is already in a state a payment cannot move it out of.
		return nil
	}

	if err := s.repo.Update(ctx, updated, prev); err != nil {
		if errors.Is(err, domain.ErrOrderStateChanged) {
			// A concurrent transition won the race; its state is authoritative.
			return nil
		}
		return err
	}
	slog.Info("order status updated from payment outcome",
		"order_id", orderID,
		"outcome", string(outcome),
		"from", string(prev),
		"to", string(updated.Status),
		"reason", reason,
	)
	return nil
}
