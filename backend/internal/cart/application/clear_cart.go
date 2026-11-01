package application

import (
	"context"
	"minimart/internal/cart/domain"
	"minimart/pkg/metrics"
)

// ClearCartRequest is the input to ClearCart.
type ClearCartRequest struct {
	UserID string
}

// ClearCart removes all items from the cart.
func (s *Service) ClearCart(ctx context.Context, req ClearCartRequest) (domain.Cart, error) {
	cart, err := s.clearCart(ctx, req)
	if err != nil {
		s.m.IncCartCleared(metrics.StatusError)
		return domain.Cart{}, err
	}
	s.m.IncCartCleared(metrics.StatusSuccess)
	return cart, nil
}

func (s *Service) clearCart(ctx context.Context, req ClearCartRequest) (domain.Cart, error) {
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
	if err := cart.Clear(); err != nil {
		return domain.Cart{}, err
	}
	if err := s.repo.Update(ctx, cart, cart.Version); err != nil {
		return domain.Cart{}, err
	}
	return cart, nil
}
