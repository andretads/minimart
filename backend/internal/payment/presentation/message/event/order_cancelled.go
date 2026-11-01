package event

import (
	"context"
	"encoding/json"
	"minimart/pkg/message"
)

// OrderCancelled is the input adapter for the order-cancelled topic: it
// refunds the payments captured for the cancelled order, because a cancelled
// order must not keep the customer's money.
func (h *Handler) OrderCancelled(ctx context.Context, msg []byte) error {
	var event message.OrderCancelled
	if err := json.Unmarshal(msg, &event); err != nil {
		return err
	}
	return h.svc.RefundOrderPayments(ctx, event.OrderID)
}
