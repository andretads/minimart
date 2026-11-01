// Package event holds the catalog domain's message input adapters. A handler here
// is structurally identical to an HTTP controller: it deserializes a transport
// payload (JSON bytes off a Kafka topic) into a call on the application service.
// It knows nothing about Kafka.
package event

import (
	"context"
	"encoding/json"
	"minimart/pkg/message"
)

// OrderStockRelease is the input adapter for the order-stock-release topic. It
// releases the order's stock reservation. Release is idempotent, so an
// at-least-once redelivery is safe.
func (h *Handler) OrderStockRelease(ctx context.Context, msg []byte) error {
	var event message.OrderStockRelease
	if err := json.Unmarshal(msg, &event); err != nil {
		return err
	}
	return h.svc.ReleaseStock(ctx, event.OrderID)
}
