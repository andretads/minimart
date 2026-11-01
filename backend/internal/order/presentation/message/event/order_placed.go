// Package event holds the order domain's own message input adapters. Consuming an
// event your own domain published is a normal pattern - here it stands in for a
// follow-up step like starting fulfilment.
package event

import (
	"context"
	"encoding/json"
	"log/slog"
	"minimart/pkg/message"
)

// OrderConfirmation reacts to the order-placed topic. It logs a confirmation; a
// real system might reserve stock or start fulfilment.
func (h *Handler) OrderConfirmation(_ context.Context, msg []byte) error {
	var event message.OrderPlaced
	if err := json.Unmarshal(msg, &event); err != nil {
		return err
	}
	slog.Info("order confirmed",
		"order_id", event.OrderID,
		"user_id", event.UserID,
		"total_cents", event.TotalCents,
		"item_count", event.ItemCount,
	)
	return nil
}
