package module

import (
	"minimart/internal/order/domain"
	"minimart/internal/order/presentation/module/client"
)

// toClient maps a domain Order to the published client type.
func (o *Order) toClient(ord domain.Order) client.Order {
	return client.Order{
		ID:                  ord.ID,
		UserID:              ord.UserID,
		CustomerID:          ord.CustomerID,
		Status:              string(ord.Status),
		TotalCents:          ord.TotalCents,
		CreatedAt:           ord.CreatedAt,
		ShippingQuoteID:     ord.ShippingQuoteID,
		ShippingMethodID:    ord.ShippingMethodID,
		ShippingMethodName:  ord.ShippingMethodName,
		ShippingCostCents:   ord.ShippingCostCents,
		DestinationSnapshot: ord.DestinationSnapshot,
	}
}
