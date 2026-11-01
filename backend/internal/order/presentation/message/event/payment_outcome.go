package event

import (
	"context"
	"encoding/json"
	"minimart/internal/order/application"
	"minimart/internal/order/domain"
	"minimart/pkg/kafka"
	"minimart/pkg/message"
)

// PaymentSucceeded reacts to the payment-succeeded topic by moving the order to
// paid. It is idempotent, so a redelivered event writes nothing and returns nil.
func (h *Handler) PaymentSucceeded(ctx context.Context, msg []byte) error {
	var event message.PaymentSucceeded
	if err := json.Unmarshal(msg, &event); err != nil {
		return err
	}
	err := h.svc.ApplyPaymentOutcome(ctx, event.OrderID, application.PaymentOutcomeSucceeded, "")
	return kafka.PermanentIf(err, domain.ErrOrderNotFound)
}

// PaymentFailed reacts to the payment-failed topic by moving the order to
// payment_failed.
func (h *Handler) PaymentFailed(ctx context.Context, msg []byte) error {
	var event message.PaymentFailed
	if err := json.Unmarshal(msg, &event); err != nil {
		return err
	}
	err := h.svc.ApplyPaymentOutcome(ctx, event.OrderID, application.PaymentOutcomeFailed, event.Reason)
	return kafka.PermanentIf(err, domain.ErrOrderNotFound)
}

// PaymentUnknown reacts to the payment-unknown topic by moving the order to
// payment_unknown (a provider timeout or technical error, to be reconciled).
func (h *Handler) PaymentUnknown(ctx context.Context, msg []byte) error {
	var event message.PaymentUnknown
	if err := json.Unmarshal(msg, &event); err != nil {
		return err
	}
	err := h.svc.ApplyPaymentOutcome(ctx, event.OrderID, application.PaymentOutcomeUnknown, event.Reason)
	return kafka.PermanentIf(err, domain.ErrOrderNotFound)
}
