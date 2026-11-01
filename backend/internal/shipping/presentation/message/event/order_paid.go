package event

import (
	"context"
	"encoding/json"
	"log/slog"
	"minimart/internal/shipping/domain"
	"minimart/pkg/kafka"
	"minimart/pkg/message"
)

func (h *Handler) OrderPaid(ctx context.Context, msg []byte) error {
	var event message.OrderPaid
	if err := json.Unmarshal(msg, &event); err != nil {
		return err
	}

	slog.Info("shipping: received order-paid event",
		"order_id", event.OrderID,
		"user_id", event.UserID,
		"customer_id", event.CustomerID,
		"quote_id", event.ShippingQuoteID,
		"method_id", event.ShippingMethodID,
	)

	shipment, err := h.svc.CreateShipmentFromOrderPaid(ctx,
		event.OrderID,
		event.UserID,
		event.CustomerID,
		event.ShippingQuoteID,
		event.ShippingMethodID,
		event.DestinationSnapshot,
		event.TotalCents,
	)
	if err != nil {
		slog.Error("shipping: failed to create shipment from order-paid",
			"order_id", event.OrderID,
			"error", err,
		)
		return kafka.PermanentIf(err,
			domain.ErrQuoteNotFound,
			domain.ErrQuoteExpired,
			domain.ErrQuoteConsumed,
			domain.ErrOwnershipMismatch,
			domain.ErrQuoteDestinationMismatch,
			domain.ErrMethodNotFound,
			domain.ErrMethodInactive,
			domain.ErrInvalidInput,
			domain.ErrInvalidAddress,
			domain.ErrInvalidTransition,
			domain.ErrInvalidQuote,
			domain.ErrInvalidQuoteDestination,
			domain.ErrOrderNotFound,
			domain.ErrOrderNotPaid,
			domain.ErrOrderCancelled,
			domain.ErrQuoteOwnershipMismatch,
			domain.ErrQuoteDestinationMismatch,
		)
	}

	slog.Info("shipping: shipment created from order-paid",
		"shipment_id", shipment.ID,
		"order_id", event.OrderID,
	)

	return nil
}
