package application

import (
	"context"
	"minimart/internal/shipping/domain"
)

// resolveRate finds the rate that prices a method for a destination. The first
// zone that matches the destination wins, and the first rate configured for
// that zone and method is the one that applies.
func (s *Service) resolveRate(ctx context.Context, dest domain.DestinationSnapshot, methodID string) (domain.ShippingRate, error) {
	zones, err := s.repo.ListZones(ctx)
	if err != nil {
		return domain.ShippingRate{}, err
	}

	var matchedZone *domain.ShippingZone
	for i := range zones {
		if zones[i].MatchesDestination(dest.Country, dest.State, dest.City, dest.Zip) {
			matchedZone = &zones[i]
			break
		}
	}
	if matchedZone == nil {
		return domain.ShippingRate{}, domain.ErrNoMatchingRate
	}

	rates, err := s.repo.ListRates(ctx, matchedZone.ID, methodID)
	if err != nil {
		return domain.ShippingRate{}, err
	}
	if len(rates) == 0 {
		return domain.ShippingRate{}, domain.ErrNoMatchingRate
	}

	rate := rates[0]
	if !rate.Active {
		return domain.ShippingRate{}, domain.ErrRateInactive
	}
	return rate, nil
}
