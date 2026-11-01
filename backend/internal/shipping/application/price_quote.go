package application

import (
	"context"
	"minimart/internal/shipping/domain"
	"minimart/pkg/metrics"
)

func (s *Service) PriceQuote(ctx context.Context, quoteID, customerID string, subtotalCents int64) (domain.ShippingQuote, error) {
	if quoteID == "" || customerID == "" || subtotalCents < 0 {
		s.m.IncShippingQuotePrice(metrics.StatusError)
		return domain.ShippingQuote{}, domain.ErrInvalidQuote
	}

	quote, err := s.repo.GetQuote(ctx, quoteID)
	if err != nil {
		s.m.IncShippingQuotePrice(metrics.StatusError)
		return domain.ShippingQuote{}, domain.ErrQuoteNotFound
	}
	if quote.CustomerID != customerID {
		s.m.IncShippingQuotePrice(metrics.StatusError)
		return domain.ShippingQuote{}, domain.ErrOwnershipMismatch
	}
	if err = quote.Validate(quote.DestinationSnapshot); err != nil {
		s.m.IncShippingQuotePrice(metrics.StatusError)
		return domain.ShippingQuote{}, err
	}

	rate, err := s.resolveRate(ctx, quote.DestinationSnapshot, quote.MethodID)
	if err != nil {
		s.m.IncShippingQuotePrice(metrics.StatusError)
		return domain.ShippingQuote{}, err
	}

	if err := quote.ApplyPricing(rate.CalculateCost(subtotalCents), subtotalCents, rate.FreeThresholdCents); err != nil {
		s.m.IncShippingQuotePrice(metrics.StatusError)
		return domain.ShippingQuote{}, err
	}

	if err := s.repo.UpdateQuote(ctx, quote); err != nil {
		s.m.IncShippingQuotePrice(metrics.StatusError)
		return domain.ShippingQuote{}, err
	}

	s.m.IncShippingQuotePrice(metrics.StatusSuccess)
	return quote, nil
}
