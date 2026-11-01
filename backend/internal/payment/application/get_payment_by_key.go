package application

import (
	"context"
	"minimart/internal/payment/domain"
)

// GetPaymentByKey returns the payment created for an idempotency key. The
// authenticated owner must match, so a key cannot be used to read another
// customer's payment.
func (s *Service) GetPaymentByKey(ctx context.Context, key, userID string) (domain.Payment, error) {
	p, err := s.repo.GetByIdempotencyKey(ctx, key)
	if err != nil {
		return domain.Payment{}, err
	}
	if p.UserID != userID {
		return domain.Payment{}, domain.ErrPaymentNotFound
	}
	return p, nil
}
