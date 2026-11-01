package application

import (
	"context"
	"minimart/internal/catalog/domain"
	"minimart/pkg/metrics"
)

// ReserveStock reserves each requested product's quantity for an order. It is
// called synchronously when an order is placed, so a sold-out product is rejected
// before the order exists. The reservation is all-or-nothing and idempotent:
// re-reserving the same order writes nothing twice.
func (s *Service) ReserveStock(ctx context.Context, orderID string, items []domain.ReservationItem) error {
	if err := s.repo.ReserveStock(ctx, orderID, items); err != nil {
		s.m.IncStockReserved(metrics.StatusError)
		return err
	}
	s.m.IncStockReserved(metrics.StatusSuccess)
	return nil
}
