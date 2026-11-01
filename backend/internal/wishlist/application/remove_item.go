package application

import (
	"context"
	"minimart/internal/wishlist/domain"
	"minimart/pkg/metrics"
)

// RemoveItemRequest is the input to RemoveItem.
type RemoveItemRequest struct {
	UserID    string
	ProductID string
}

// RemoveItem removes a product from the wishlist.
func (s *Service) RemoveItem(ctx context.Context, req RemoveItemRequest) (domain.Wishlist, error) {
	wishlist, err := s.removeItem(ctx, req)
	if err != nil {
		s.m.IncWishlistItemRemoved(metrics.StatusError)
		return domain.Wishlist{}, err
	}
	s.m.IncWishlistItemRemoved(metrics.StatusSuccess)
	return wishlist, nil
}

func (s *Service) removeItem(ctx context.Context, req RemoveItemRequest) (domain.Wishlist, error) {
	if req.UserID == "" {
		return domain.Wishlist{}, ErrUserIDRequired
	}
	if req.ProductID == "" {
		return domain.Wishlist{}, ErrProductIDRequired
	}
	cust, err := s.modules.EnsureCustomer(ctx, req.UserID)
	if err != nil {
		return domain.Wishlist{}, err
	}
	wishlist, err := s.repo.GetByCustomerID(ctx, cust.ID)
	if err != nil {
		return domain.Wishlist{}, err
	}
	if err := wishlist.RemoveItem(req.ProductID); err != nil {
		return domain.Wishlist{}, err
	}
	if err := s.repo.Update(ctx, wishlist); err != nil {
		return domain.Wishlist{}, err
	}
	return wishlist, nil
}
