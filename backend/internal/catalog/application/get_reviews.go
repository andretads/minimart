package application

import (
	"context"
	"minimart/internal/catalog/domain"
)

// GetReviews returns all approved reviews for a product.
func (s *Service) GetReviews(ctx context.Context, productID string) ([]domain.Review, error) {
	if _, err := s.repo.GetByID(ctx, productID); err != nil {
		return nil, err
	}
	return s.repo.GetReviews(ctx, productID)
}

// ListReviewsRequest carries pagination parameters for product reviews.
type ListReviewsRequest struct {
	ProductID string
	Page      int
	PageSize  int
}

// ListReviewsResult wraps a page of reviews with pagination metadata.
type ListReviewsResult struct {
	Items      []domain.Review
	Total      int
	Page       int
	PageSize   int
	TotalPages int
}

// ListReviews returns a page of approved reviews for a product, newest first.
func (s *Service) ListReviews(ctx context.Context, req ListReviewsRequest) (ListReviewsResult, error) {
	if req.Page < 1 {
		return ListReviewsResult{}, domain.ErrInvalidPage
	}
	if req.PageSize < 1 || req.PageSize > 100 {
		return ListReviewsResult{}, domain.ErrInvalidPageSize
	}
	offset := (req.Page - 1) * req.PageSize
	items, err := s.repo.GetReviewsPaginated(ctx, req.ProductID, req.PageSize, offset)
	if err != nil {
		return ListReviewsResult{}, err
	}
	total, err := s.repo.CountReviews(ctx, req.ProductID)
	if err != nil {
		return ListReviewsResult{}, err
	}
	totalPages := (total + req.PageSize - 1) / req.PageSize
	return ListReviewsResult{
		Items:      items,
		Total:      total,
		Page:       req.Page,
		PageSize:   req.PageSize,
		TotalPages: totalPages,
	}, nil
}
