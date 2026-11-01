package application

import (
	"context"
	"minimart/pkg/metrics"
)

// ReleaseStock returns an order's outstanding stock reservation. It is called
// when an order will not be fulfilled (it was cancelled or its payment failed).
// It is idempotent, so a redelivered release is safe.
func (s *Service) ReleaseStock(ctx context.Context, orderID string) error {
	if err := s.repo.ReleaseStock(ctx, orderID); err != nil {
		s.m.IncStockReleased(metrics.StatusError)
		return err
	}
	s.m.IncStockReleased(metrics.StatusSuccess)
	return nil
}
