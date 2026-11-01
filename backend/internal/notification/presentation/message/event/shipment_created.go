package event

import (
	"context"
	"encoding/json"
	"minimart/internal/notification/domain"
	"minimart/pkg/kafka"
	"minimart/pkg/message"
)

// ShipmentCreated is the input adapter for the shipment-created topic.
func (h *Handler) ShipmentCreated(ctx context.Context, msg []byte) error {
	var event message.ShipmentCreated
	if err := json.Unmarshal(msg, &event); err != nil {
		return err
	}
	err := h.svc.SendShipmentCreated(ctx, event.OrderID, event.UserID, event.ShippingCostCents)
	return kafka.PermanentIf(err,
		domain.ErrEmptyID,
		domain.ErrEmptyUserID,
		domain.ErrEmptyOrderID,
		domain.ErrNonPositiveTotal,
		domain.ErrNegativeShippingCost,
		domain.ErrInvalidChannel,
	)
}
