package application

import (
	"context"
	"minimart/pkg/metrics"
)

func (s *Service) ExpireQuotes(ctx context.Context) error {
	if err := s.repo.ExpireQuotes(ctx); err != nil {
		s.m.IncShippingQuotesExpire(metrics.StatusError)
		return err
	}
	s.m.IncShippingQuotesExpire(metrics.StatusSuccess)
	return nil
}
