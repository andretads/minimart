package event

import (
	"context"
	"encoding/json"
	"minimart/internal/notification/domain"
	"minimart/pkg/kafka"
	"minimart/pkg/message"
)

// PaymentFailed is the input adapter for the payment-failed topic.
func (h *Handler) PaymentFailed(ctx context.Context, msg []byte) error {
	var event message.PaymentFailed
	if err := json.Unmarshal(msg, &event); err != nil {
		return err
	}
	err := h.svc.SendPaymentFailed(ctx, event.OrderID, event.UserID, event.AmountCents, event.Reason)
	return kafka.PermanentIf(err,
		domain.ErrEmptyID,
		domain.ErrEmptyUserID,
		domain.ErrEmptyOrderID,
		domain.ErrNonPositiveTotal,
		domain.ErrInvalidChannel,
	)
}
