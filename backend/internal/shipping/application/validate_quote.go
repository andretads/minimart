package application

import (
	"context"
	"minimart/internal/shipping/domain"
	"minimart/pkg/metrics"
)

func (s *Service) ValidateQuote(ctx context.Context, quoteID, customerID string) (domain.ShippingQuote, error) {
	quote, err := s.repo.GetQuote(ctx, quoteID)
	if err != nil {
		s.m.IncShippingQuoteValidate(metrics.StatusError)
		return domain.ShippingQuote{}, domain.ErrQuoteNotFound
	}
	if quote.CustomerID != customerID {
		s.m.IncShippingQuoteValidate(metrics.StatusError)
		return domain.ShippingQuote{}, domain.ErrOwnershipMismatch
	}
	if err := quote.Validate(quote.DestinationSnapshot); err != nil {
		s.m.IncShippingQuoteValidate(metrics.StatusError)
		return domain.ShippingQuote{}, err
	}
	s.m.IncShippingQuoteValidate(metrics.StatusSuccess)
	return quote, nil
}
