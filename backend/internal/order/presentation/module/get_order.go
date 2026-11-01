package module

import (
	"context"
	"minimart/internal/order/presentation/module/client"
)

// GetOrder is the cross-domain entry point. It calls the service, then maps the
// domain result and error into the published client types, so callers depend only
// on the client package.
func (o *Order) GetOrder(ctx context.Context, id string) (client.Order, error) {
	ord, err := o.svc.GetOrder(ctx, id)
	if err != nil {
		return client.Order{}, o.translateError(err)
	}
	return o.toClient(ord), nil
}
