package application

import (
	"context"
	"minimart/internal/cart/domain"
	"minimart/pkg/metrics"
)

// GetCartRequest is the input to GetCart.
type GetCartRequest struct {
	UserID string
}

// GetCart retrieves the customer's cart without creating one.
func (s *Service) GetCart(ctx context.Context, req GetCartRequest) (domain.Cart, error) {
	cart, err := s.getCart(ctx, req)
	if err != nil {
		s.m.IncCartRetrieved(metrics.StatusError)
		return domain.Cart{}, err
	}
	s.m.IncCartRetrieved(metrics.StatusSuccess)
	return cart, nil
}

func (s *Service) getCart(ctx context.Context, req GetCartRequest) (domain.Cart, error) {
	if req.UserID == "" {
		return domain.Cart{}, ErrUserIdRequired
	}
	cust, err := s.modules.GetCustomerByUserID(ctx, req.UserID)
	if err != nil {
		return domain.Cart{}, err
	}
	return s.repo.GetByCustomerID(ctx, cust.ID)
}
