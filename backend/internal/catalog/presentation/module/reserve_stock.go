package module

import (
	"context"
	"errors"
	"minimart/internal/catalog/domain"
	"minimart/internal/catalog/presentation/module/client"
)

// ReserveStock adapts the application use case to the published contract,
// translating domain types and errors into their client-level equivalents.
func (c *Catalog) ReserveStock(ctx context.Context, orderID string, items []client.LineItem) error {
	domainItems := make([]domain.ReservationItem, 0, len(items))
	for _, it := range items {
		domainItems = append(domainItems, domain.ReservationItem{ProductID: it.ProductID, Quantity: it.Quantity})
	}
	err := c.svc.ReserveStock(ctx, orderID, domainItems)
	switch {
	case err == nil:
		return nil
	case errors.Is(err, domain.ErrInsufficientStock):
		return client.ErrInsufficientStock
	case errors.Is(err, domain.ErrProductNotFound):
		return client.ErrProductNotFound
	default:
		return err
	}
}
