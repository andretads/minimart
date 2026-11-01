package application

import (
	"context"
	"errors"
	"minimart/internal/cart/domain"
	"minimart/pkg/metrics"
)

// AddItemRequest is the input to AddItem.
type AddItemRequest struct {
	UserID    string
	ProductID string
	Quantity  int
}

// AddItem adds an item to the cart or increments quantity if already present.
func (s *Service) AddItem(ctx context.Context, req AddItemRequest) (domain.Cart, error) {
	cart, err := s.addItem(ctx, req)
	if err != nil {
		s.m.IncCartItemAdded(metrics.StatusError)
		return domain.Cart{}, err
	}
	s.m.IncCartItemAdded(metrics.StatusSuccess)
	return cart, nil
}

func (s *Service) addItem(ctx context.Context, req AddItemRequest) (domain.Cart, error) {
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
		cart, err = domain.NewCart(newID(), cust.ID)
		if err != nil {
			return domain.Cart{}, err
		}
		createErr := s.repo.Create(ctx, cart)
		if createErr != nil {
			if errors.Is(createErr, domain.ErrCartAlreadyExists) {
				cart, err = s.repo.GetByCustomerID(ctx, cust.ID)
				if err != nil {
					return domain.Cart{}, err
				}
			} else {
				return domain.Cart{}, createErr
			}
		}
	}
	prod, err := s.modules.GetProduct(ctx, req.ProductID)
	if err != nil {
		return domain.Cart{}, err
	}
	if err := cart.AddItem(prod.ID, prod.Name, prod.PriceCents, req.Quantity); err != nil {
		return domain.Cart{}, err
	}
	if err := s.repo.Update(ctx, cart, cart.Version); err != nil {
		return domain.Cart{}, err
	}
	return cart, nil
}
