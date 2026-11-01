package application

import (
	"context"
	"minimart/internal/auth/domain"
)

// GetAccount retrieves an account by its ID.
func (s *Service) GetAccount(ctx context.Context, id string) (domain.Account, error) {
	return s.repo.GetByID(ctx, id)
}
