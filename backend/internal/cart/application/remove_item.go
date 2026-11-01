package application

import (
	"context"
	"minimart/internal/cart/domain"
	"minimart/pkg/metrics"
)

// RemoveItemRequest is the input to RemoveItem.
type RemoveItemRequest struct {
	UserID    string
	ProductID string
}

// RemoveItem removes a specific product from the cart.
func (s *Service) RemoveItem(ctx context.Context, req RemoveItemRequest) (domain.Cart, error) {
	cart, err := s.removeItem(ctx, req)
	if err != nil {
		s.m.IncCartItemRemoved(metrics.StatusError)
		return domain.Cart{}, err
	}
	s.m.IncCartItemRemoved(metrics.StatusSuccess)
	return cart, nil
}

func (s *Service) removeItem(ctx context.Context, req RemoveItemRequest) (domain.Cart, error) {
	if req.UserID == "" {
		return domain.Cart{}, ErrUserIdRequired
	}
	cust, err := s.modules.EnsureCustomer(ctx, req.UserID)
	if err != nil {
		return domain.Cart{}, err
	}
	cart, err := s.repo.GetByCustomerID(ctx, cust.ID)
	if err != nil {
		return domain.Cart{}, err
	}
	if err := cart.RemoveItem(req.ProductID); err != nil {
		return domain.Cart{}, err
	}
	if err := s.repo.Update(ctx, cart, cart.Version); err != nil {
		return domain.Cart{}, err
	}
	return cart, nil
}
