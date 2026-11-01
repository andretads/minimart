package event

import (
	"context"
	"encoding/json"
	"minimart/internal/notification/domain"
	"minimart/pkg/kafka"
	"minimart/pkg/message"
)

// PaymentSucceeded is the input adapter for the payment-succeeded topic.
func (h *Handler) PaymentSucceeded(ctx context.Context, msg []byte) error {
	var event message.PaymentSucceeded
	if err := json.Unmarshal(msg, &event); err != nil {
		return err
	}
	err := h.svc.SendPaymentSucceeded(ctx, event.OrderID, event.UserID, event.AmountCents)
	return kafka.PermanentIf(err,
		domain.ErrEmptyID,
		domain.ErrEmptyUserID,
		domain.ErrEmptyOrderID,
		domain.ErrNonPositiveTotal,
		domain.ErrInvalidChannel,
	)
}
