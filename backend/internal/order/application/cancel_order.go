package application

import (
	"context"
	"minimart/internal/order/domain"
	"minimart/pkg/metrics"
)

// CancelOrder loads the order, marks it cancelled through the domain guard, and
// persists the new state. The repository writes the update and the OrderCancelled
// event to the transactional outbox in the same transaction. Cancelling an
// already-cancelled order (or one that does not exist) is reported as an error.
func (s *Service) CancelOrder(ctx context.Context, id string) (domain.Order, error) {
	order, err := s.cancelOrder(ctx, id)
	if err != nil {
		s.m.IncOrderCancelled(metrics.StatusError)
		return order, err
	}
	s.m.IncOrderCancelled(metrics.StatusSuccess)
	return order, nil
}

func (s *Service) cancelOrder(ctx context.Context, id string) (domain.Order, error) {
	order, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return domain.Order{}, err
	}
	cancelled, err := order.Cancel()
	if err != nil {
		return domain.Order{}, err
	}
	if err := s.repo.Update(ctx, cancelled, order.Status); err != nil {
		return domain.Order{}, err
	}
	return cancelled, nil
}
