package module

import (
	"context"
	"errors"
	"minimart/internal/auth/domain"

	client "minimart/internal/auth/presentation/module/client"
)

// GetAccountEmail is the cross-domain entry point. It calls the service, then maps the
// domain result into the published client types, so callers depend only on the client package.
func (a *Auth) GetAccountEmail(ctx context.Context, userID string) (string, error) {
	acc, err := a.svc.GetAccount(ctx, userID)
	if err != nil {
		if errors.Is(err, domain.ErrAccountNotFound) {
			return "", client.ErrAccountNotFound
		}
		return "", err
	}
	if acc.Email == "" {
		return "", client.ErrEmailNotFound
	}
	return acc.Email, nil
}
