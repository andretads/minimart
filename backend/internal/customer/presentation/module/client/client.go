package client

import (
	"context"
	"errors"
	"time"
)

var ErrCustomerNotFound = errors.New("customer: cliente não encontrado")

type Customer struct {
	ID          string
	UserID      string
	DisplayName string
	Status      string
	CreatedAt   time.Time
}

type CustomerClient interface {
	EnsureCustomer(ctx context.Context, userID string) (Customer, error)
	GetCustomerByID(ctx context.Context, id string) (Customer, error)
	GetCustomerByUserID(ctx context.Context, userID string) (Customer, error)
}
