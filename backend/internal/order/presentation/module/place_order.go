package module

import (
	"context"
	"minimart/internal/order/application"
	"minimart/internal/order/presentation/module/client"
)

// PlaceOrder is the cross-domain entry point. It calls the service, then maps the
// domain result and error into the published client types, so callers depend only
// on the client package.
func (o *Order) PlaceOrder(ctx context.Context, req client.PlaceOrderRequest) (client.Order, error) {
	appReq := application.PlaceOrderRequest{
		UserID:              req.UserID,
		IdempotencyKey:      req.IdempotencyKey,
		Items:               make([]application.PlaceOrderItem, len(req.Items)),
		ShippingQuoteID:     req.ShippingQuoteID,
		ShippingMethodID:    req.ShippingMethodID,
		ShippingMethodName:  req.ShippingMethodName,
		DestinationSnapshot: req.DestinationSnapshot,
		ShippingCostCents:   req.ShippingCostCents,
	}
	for i, item := range req.Items {
		appReq.Items[i] = application.PlaceOrderItem{
			ProductID: item.ProductID,
			Quantity:  item.Quantity,
		}
	}

	ord, err := o.svc.PlaceOrder(ctx, appReq)
	if err != nil {
		return client.Order{}, o.translateError(err)
	}
	return o.toClient(ord), nil
}
