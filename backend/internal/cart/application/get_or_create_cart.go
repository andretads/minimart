package application

import (
	"context"
	"errors"
	"minimart/internal/cart/domain"
	"minimart/pkg/metrics"
)

// GetOrCreateCartRequest is the input to GetOrCreateCart.
type GetOrCreateCartRequest struct {
	UserID string
}

// GetOrCreateCart retrieves the customer's active cart or creates one if none exists.
func (s *Service) GetOrCreateCart(ctx context.Context, req GetOrCreateCartRequest) (domain.Cart, error) {
	cart, err := s.getOrCreateCart(ctx, req)
	if err != nil {
		s.m.IncCartCreated(metrics.StatusError)
		return domain.Cart{}, err
	}
	s.m.IncCartCreated(metrics.StatusSuccess)
	return cart, nil
}

func (s *Service) getOrCreateCart(ctx context.Context, req GetOrCreateCartRequest) (domain.Cart, error) {
	if req.UserID == "" {
		return domain.Cart{}, ErrUserIdRequired
	}
	cust, err := s.modules.EnsureCustomer(ctx, req.UserID)
	if err != nil {
		return domain.Cart{}, err
	}
	cart, err := s.repo.GetByCustomerID(ctx, cust.ID)
	if err != nil {
		if !errors.Is(err, domain.ErrCartNotFound) {
			return domain.Cart{}, err
		}
		id := newID()
		newCart, err := domain.NewCart(id, cust.ID)
		if err != nil {
			return domain.Cart{}, err
		}
		if err := s.repo.Create(ctx, newCart); err != nil {
			if errors.Is(err, domain.ErrCartAlreadyExists) {
				return s.repo.GetByCustomerID(ctx, cust.ID)
			}
			return domain.Cart{}, err
		}
		return newCart, nil
	}
	return cart, nil
}
