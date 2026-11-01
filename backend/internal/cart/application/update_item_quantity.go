package application

import (
	"context"
	"minimart/internal/cart/domain"
	"minimart/pkg/metrics"
)

// UpdateItemQuantityRequest is the input to UpdateItemQuantity.
type UpdateItemQuantityRequest struct {
	UserID    string
	ProductID string
	Quantity  int
}

// UpdateItemQuantity sets the quantity for a specific product in the cart.
func (s *Service) UpdateItemQuantity(ctx context.Context, req UpdateItemQuantityRequest) (domain.Cart, error) {
	cart, err := s.updateItemQuantity(ctx, req)
	if err != nil {
		s.m.IncCartItemUpdated(metrics.StatusError)
		return domain.Cart{}, err
	}
	s.m.IncCartItemUpdated(metrics.StatusSuccess)
	return cart, nil
}

func (s *Service) updateItemQuantity(ctx context.Context, req UpdateItemQuantityRequest) (domain.Cart, error) {
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
	if err := cart.UpdateItemQuantity(req.ProductID, req.Quantity); err != nil {
		return domain.Cart{}, err
	}
	if err := s.repo.Update(ctx, cart, cart.Version); err != nil {
		return domain.Cart{}, err
	}
	return cart, nil
}
