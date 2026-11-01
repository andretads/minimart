package module

import (
	"context"
	"errors"
	"minimart/internal/category/domain"
	"minimart/internal/category/presentation/module/client"
)

// GetCategory is the cross-domain entry point. It calls the service, then maps the
// domain result and error into the published client types, so callers depend only
// on the client package.
func (c *Category) GetCategory(ctx context.Context, id string) (client.CategoryDTO, error) {
	cat, err := c.svc.GetCategory(ctx, id)
	if err != nil {
		if errors.Is(err, domain.ErrCategoryNotFound) {
			return client.CategoryDTO{}, client.ErrCategoryNotFound
		}
		return client.CategoryDTO{}, err
	}
	return toClient(cat), nil
}
