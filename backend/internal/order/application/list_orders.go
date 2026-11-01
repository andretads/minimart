package application

import (
	"context"
	"minimart/internal/order/domain"
	"strings"
)

// ListOrdersRequest carries the user whose orders are listed and the pagination
// window. Page is 1-based; PageSize is capped at 100.
type ListOrdersRequest struct {
	UserID   string
	Page     int
	PageSize int
}

// ListOrdersResult wraps one page of orders with pagination metadata.
type ListOrdersResult struct {
	Items      []domain.Order
	Total      int
	Page       int
	PageSize   int
	TotalPages int
}

// ListOrders returns one page of a user's orders, newest first.
func (s *Service) ListOrders(ctx context.Context, req ListOrdersRequest) (ListOrdersResult, error) {
	if strings.TrimSpace(req.UserID) == "" {
		return ListOrdersResult{}, domain.ErrUserIDRequired
	}
	if req.Page < 1 {
		return ListOrdersResult{}, domain.ErrInvalidPage
	}
	if req.PageSize < 1 || req.PageSize > 100 {
		return ListOrdersResult{}, domain.ErrInvalidPageSize
	}

	offset := (req.Page - 1) * req.PageSize
	items, err := s.repo.ListByUserID(ctx, req.UserID, req.PageSize, offset)
	if err != nil {
		return ListOrdersResult{}, err
	}
	total, err := s.repo.CountByUserID(ctx, req.UserID)
	if err != nil {
		return ListOrdersResult{}, err
	}

	totalPages := (total + req.PageSize - 1) / req.PageSize
	return ListOrdersResult{
		Items:      items,
		Total:      total,
		Page:       req.Page,
		PageSize:   req.PageSize,
		TotalPages: totalPages,
	}, nil
}
