package module

import (
	"errors"
	"minimart/internal/order/application"
	"minimart/internal/order/domain"
	"minimart/internal/order/presentation/module/client"

	catalogclient "minimart/internal/catalog/presentation/module/client"
)

// translateError maps domain and application errors to published client errors.
func (o *Order) translateError(err error) error {
	switch {
	case errors.Is(err, domain.ErrOrderNotFound):
		return client.ErrOrderNotFound
	case errors.Is(err, domain.ErrIdempotencyConflict):
		return client.ErrIdempotencyConflict
	case errors.Is(err, application.ErrOrderInProgress):
		return client.ErrOrderInProgress
	case errors.Is(err, catalogclient.ErrInsufficientStock):
		return client.ErrInsufficientStock
	case errors.Is(err, catalogclient.ErrProductNotFound):
		return client.ErrProductNotFound
	case errors.Is(err, domain.ErrNoItems):
		return client.ErrNoItems
	case errors.Is(err, domain.ErrInvalidQuantity):
		return client.ErrInvalidQuantity
	case errors.Is(err, application.ErrIdempotencyKeyRequired):
		return client.ErrIdempotencyKeyRequired
	default:
		return err
	}
}
