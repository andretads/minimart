package module

import (
	"context"
	"errors"
	"minimart/internal/customer/domain"
	"minimart/internal/customer/presentation/module/client"
)

// EnsureCustomer is the cross-domain entry point. It calls the service, then maps the
// domain result and error into the published client types, so callers depend only
// on the client package.
func (a *Customer) EnsureCustomer(ctx context.Context, userID string) (client.Customer, error) {
	c, err := a.svc.EnsureCustomer(ctx, userID)
	if err != nil {
		if errors.Is(err, domain.ErrCustomerNotFound) {
			return client.Customer{}, client.ErrCustomerNotFound
		}
		return client.Customer{}, err
	}
	return toClient(c), nil
}
