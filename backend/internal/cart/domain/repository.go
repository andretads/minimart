package domain

import "context"

type Repository interface {
	GetByCustomerID(ctx context.Context, customerID string) (Cart, error)
	GetByCustomerIDIncludingConsumed(ctx context.Context, customerID string) (Cart, error)
	Create(ctx context.Context, cart Cart) error
	Update(ctx context.Context, cart Cart, expectedVersion int) error
}
