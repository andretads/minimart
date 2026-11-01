package application

import (
	"context"
	"minimart/internal/payment/domain"
)

// GetPayment returns a single payment by ID.
func (s *Service) GetPayment(ctx context.Context, id string) (domain.Payment, error) {
	return s.repo.GetByID(ctx, id)
}
