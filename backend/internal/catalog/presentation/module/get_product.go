package module

import (
	"context"
	"errors"
	"minimart/internal/catalog/domain"
	"minimart/internal/catalog/presentation/module/client"
)

// GetProduct is the cross-domain entry point. It calls the service, then maps the
// domain result and error into the published client types, so callers depend only
// on the client package.
func (c *Catalog) GetProduct(ctx context.Context, id string) (client.Product, error) {
	p, err := c.svc.GetProduct(ctx, id)
	if err != nil {
		if errors.Is(err, domain.ErrProductNotFound) {
			return client.Product{}, client.ErrProductNotFound
		}
		return client.Product{}, err
	}
	return client.Product{ID: p.ID, Name: p.Name, PriceCents: p.PriceCents}, nil
}
