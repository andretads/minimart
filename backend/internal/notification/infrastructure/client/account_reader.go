package client

import (
	"context"
	"errors"
	"minimart/internal/auth/presentation/module/client"
	"minimart/internal/notification/domain"
	"minimart/pkg/module/contracts"
)

type AccountReader struct {
	Contracts *contracts.Contracts
}

var _ domain.RecipientResolver = (*AccountReader)(nil)

func NewAccountReader(c *contracts.Contracts) *AccountReader {
	return &AccountReader{Contracts: c}
}

func (a *AccountReader) EmailForUser(ctx context.Context, userID string) (string, error) {
	email, err := a.Contracts.GetAccountEmail(ctx, userID)
	if err != nil {
		if errors.Is(err, client.ErrAccountNotFound) || errors.Is(err, client.ErrEmailNotFound) {
			return "", domain.ErrRecipientNotFound
		}
		return "", err
	}
	return email, nil
}

func (a *AccountReader) PhoneForUser(ctx context.Context, userID string) (string, error) {
	customer, err := a.Contracts.GetCustomerByUserID(ctx, userID)
	if err != nil {
		return "", domain.ErrRecipientNotFound
	}
	addr, err := a.Contracts.GetDefaultAddressByCustomerID(ctx, customer.ID)
	if err != nil {
		return "", domain.ErrRecipientNotFound
	}
	if addr.Phone == "" {
		return "", domain.ErrRecipientNotFound
	}
	return addr.Phone, nil
}

func (a *AccountReader) PushTokenForUser(ctx context.Context, userID string) (string, error) {
	return "", domain.ErrRecipientNotFound
}
