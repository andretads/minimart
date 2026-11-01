package application

import (
	"context"
	"minimart/internal/shipping/domain"
	"minimart/pkg/metrics"
)

func (s *Service) GetQuote(ctx context.Context, id string) (domain.ShippingQuote, error) {
	quote, err := s.repo.GetQuote(ctx, id)
	if err != nil {
		s.m.IncShippingQuoteGet(metrics.StatusError)
		return domain.ShippingQuote{}, err
	}
	s.m.IncShippingQuoteGet(metrics.StatusSuccess)
	return quote, nil
}
