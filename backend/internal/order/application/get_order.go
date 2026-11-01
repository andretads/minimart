package application

import (
	"context"
	"minimart/internal/order/domain"
)

func (s *Service) GetOrder(ctx context.Context, id string) (domain.Order, error) {
	return s.repo.GetByID(ctx, id)
}
