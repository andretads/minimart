package event

import (
	"context"
	"encoding/json"
	"minimart/internal/notification/domain"
	"minimart/pkg/kafka"
	"minimart/pkg/message"
)

// OrderCancelled is the input adapter for the order-cancelled topic.
func (h *Handler) OrderCancelled(ctx context.Context, msg []byte) error {
	var event message.OrderCancelled
	if err := json.Unmarshal(msg, &event); err != nil {
		return err
	}
	err := h.svc.SendOrderCancellation(ctx, event.OrderID, event.UserID, event.TotalCents)
	return kafka.PermanentIf(err,
		domain.ErrEmptyID,
		domain.ErrEmptyUserID,
		domain.ErrEmptyOrderID,
		domain.ErrNonPositiveTotal,
		domain.ErrInvalidChannel,
	)
}
