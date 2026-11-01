package application

import (
	"context"
	"errors"
	"minimart/internal/wishlist/domain"
	"minimart/pkg/metrics"
)

// AddItemRequest is the input to AddItem.
type AddItemRequest struct {
	UserID    string
	ProductID string
}

// AddItem adds a product to the wishlist.
func (s *Service) AddItem(ctx context.Context, req AddItemRequest) (domain.Wishlist, error) {
	wishlist, err := s.addItem(ctx, req)
	if err != nil {
		s.m.IncWishlistItemAdded(metrics.StatusError)
		return domain.Wishlist{}, err
	}
	s.m.IncWishlistItemAdded(metrics.StatusSuccess)
	return wishlist, nil
}

func (s *Service) addItem(ctx context.Context, req AddItemRequest) (domain.Wishlist, error) {
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
	// Validate product exists
	_, err = s.modules.GetProduct(ctx, req.ProductID)
	if err != nil {
		return domain.Wishlist{}, err
	}
	wishlist, err := s.repo.GetByCustomerID(ctx, cust.ID)
	if err != nil {
		if !errors.Is(err, domain.ErrWishlistNotFound) {
			return domain.Wishlist{}, err
		}
		wishlist, err = domain.NewWishlist(newID(), cust.ID)
		if err != nil {
			return domain.Wishlist{}, err
		}
		if err := s.repo.Create(ctx, wishlist); err != nil {
			if errors.Is(err, domain.ErrWishlistAlreadyExists) {
				wishlist, err = s.repo.GetByCustomerID(ctx, cust.ID)
				if err != nil {
					return domain.Wishlist{}, err
				}
			} else {
				return domain.Wishlist{}, err
			}
		}
	}
	if err := wishlist.AddItem(req.ProductID); err != nil {
		return domain.Wishlist{}, err
	}
	if err := s.repo.Update(ctx, wishlist); err != nil {
		return domain.Wishlist{}, err
	}
	return wishlist, nil
}
