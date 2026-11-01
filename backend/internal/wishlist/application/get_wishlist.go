package application

import (
	"context"
	"errors"
	"minimart/internal/wishlist/domain"
	"minimart/pkg/metrics"
)

// GetWishlistRequest is the input to GetWishlist.
type GetWishlistRequest struct {
	UserID string
}

// GetWishlist retrieves the customer's wishlist or creates one if none exists.
func (s *Service) GetWishlist(ctx context.Context, req GetWishlistRequest) (domain.Wishlist, error) {
	wishlist, err := s.getOrCreateWishlist(ctx, req)
	if err != nil {
		s.m.IncWishlistCreated(metrics.StatusError)
		return domain.Wishlist{}, err
	}
	s.m.IncWishlistCreated(metrics.StatusSuccess)
	return wishlist, nil
}

func (s *Service) getOrCreateWishlist(ctx context.Context, req GetWishlistRequest) (domain.Wishlist, error) {
	if req.UserID == "" {
		return domain.Wishlist{}, ErrUserIDRequired
	}
	cust, err := s.modules.EnsureCustomer(ctx, req.UserID)
	if err != nil {
		return domain.Wishlist{}, err
	}
	wishlist, err := s.repo.GetByCustomerID(ctx, cust.ID)
	if err != nil {
		if !errors.Is(err, domain.ErrWishlistNotFound) {
			return domain.Wishlist{}, err
		}
		newWishlist, err := domain.NewWishlist(newID(), cust.ID)
		if err != nil {
			return domain.Wishlist{}, err
		}
		if err := s.repo.Create(ctx, newWishlist); err != nil {
			if errors.Is(err, domain.ErrWishlistAlreadyExists) {
				return s.repo.GetByCustomerID(ctx, cust.ID)
			}
			return domain.Wishlist{}, err
		}
		return newWishlist, nil
	}
	return wishlist, nil
}
