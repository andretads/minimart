package application

import (
	"context"
	"errors"
	"minimart/internal/wishlist/domain"
	"minimart/pkg/metrics"
)

// ClearWishlistRequest is the input to ClearWishlist.
type ClearWishlistRequest struct {
	UserID string
}

// ClearWishlist removes all items from the wishlist.
func (s *Service) ClearWishlist(ctx context.Context, req ClearWishlistRequest) error {
	err := s.clearWishlist(ctx, req)
	if err != nil {
		s.m.IncWishlistCleared(metrics.StatusError)
		return err
	}
	s.m.IncWishlistCleared(metrics.StatusSuccess)
	return nil
}

func (s *Service) clearWishlist(ctx context.Context, req ClearWishlistRequest) error {
	if req.UserID == "" {
		return ErrUserIDRequired
	}
	cust, err := s.modules.EnsureCustomer(ctx, req.UserID)
	if err != nil {
		return err
	}
	wishlist, err := s.repo.GetByCustomerID(ctx, cust.ID)
	if err != nil {
		if errors.Is(err, domain.ErrWishlistNotFound) {
			return nil
		}
		return err
	}
	if err := wishlist.Clear(); err != nil {
		return err
	}
	return s.repo.Update(ctx, wishlist)
}
