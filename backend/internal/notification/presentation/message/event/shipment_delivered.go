package event

import (
	"context"
	"encoding/json"
	"minimart/internal/notification/domain"
	"minimart/pkg/kafka"
	"minimart/pkg/message"
)

// ShipmentDelivered is the input adapter for the shipment-delivered topic.
func (h *Handler) ShipmentDelivered(ctx context.Context, msg []byte) error {
	var event message.ShipmentDelivered
	if err := json.Unmarshal(msg, &event); err != nil {
		return err
	}
	err := h.svc.SendShipmentDelivered(ctx, event.OrderID, event.UserID)
	return kafka.PermanentIf(err,
		domain.ErrEmptyID,
		domain.ErrEmptyUserID,
		domain.ErrEmptyOrderID,
		domain.ErrInvalidChannel,
	)
}
