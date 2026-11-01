package event

import (
	"context"
	"encoding/json"
	"minimart/internal/notification/domain"
	"minimart/pkg/kafka"
	"minimart/pkg/message"
)

// ShipmentShipped is the input adapter for the shipment-shipped topic.
func (h *Handler) ShipmentShipped(ctx context.Context, msg []byte) error {
	var event message.ShipmentShipped
	if err := json.Unmarshal(msg, &event); err != nil {
		return err
	}
	err := h.svc.SendShipmentShipped(ctx, event.OrderID, event.UserID, event.TrackingCode, event.Carrier)
	return kafka.PermanentIf(err,
		domain.ErrEmptyID,
		domain.ErrEmptyUserID,
		domain.ErrEmptyOrderID,
		domain.ErrInvalidChannel,
	)
}
