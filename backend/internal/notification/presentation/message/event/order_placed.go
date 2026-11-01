// Package event holds the notification domain's message input adapters. A handler
// here is structurally identical to an HTTP controller: it deserializes a transport
// payload (JSON bytes off a Kafka topic) into a request and calls the application
// service. It knows nothing about Kafka.
package event

import (
	"context"
	"encoding/json"
	"minimart/internal/notification/domain"
	"minimart/pkg/kafka"
	"minimart/pkg/message"
)

// OrderPlaced is the input adapter for the order-placed topic.
func (h *Handler) OrderPlaced(ctx context.Context, msg []byte) error {
	var event message.OrderPlaced
	if err := json.Unmarshal(msg, &event); err != nil {
		return err
	}
	err := h.svc.SendOrderConfirmation(ctx, event.OrderID, event.UserID, event.TotalCents)
	return kafka.PermanentIf(err,
		domain.ErrEmptyID,
		domain.ErrEmptyUserID,
		domain.ErrEmptyOrderID,
		domain.ErrNonPositiveTotal,
		domain.ErrNegativeShippingCost,
		domain.ErrInvalidChannel,
	)
}
