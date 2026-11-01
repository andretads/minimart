package application

import (
	"context"
	"minimart/internal/catalog/domain"
	"minimart/pkg/metrics"
	"strings"
	"time"

	"github.com/google/uuid"
)

// AddReviewRequest is the input to the AddReview use case.
type AddReviewRequest struct {
	ProductID   string
	CustomerID  string
	Description string
	Rating      int
}

// AddReview adds a new review for a product.
func (s *Service) AddReview(ctx context.Context, req AddReviewRequest) (domain.Review, error) {
	if req.Rating < 1 || req.Rating > 5 {
		s.m.IncReviewAdded(metrics.StatusError)
		return domain.Review{}, domain.ErrInvalidRating
	}
	desc := strings.TrimSpace(req.Description)
	if desc == "" {
		s.m.IncReviewAdded(metrics.StatusError)
		return domain.Review{}, domain.ErrEmptyName // reuse empty name error for empty description
	}
	if req.CustomerID == "" {
		s.m.IncReviewAdded(metrics.StatusError)
		return domain.Review{}, domain.ErrEmptyCustomerID
	}

	// Verify product exists
	_, err := s.repo.GetByID(ctx, req.ProductID)
	if err != nil {
		s.m.IncReviewAdded(metrics.StatusError)
		return domain.Review{}, err
	}

	review := domain.Review{
		ID:          uuid.New().String(),
		ProductID:   req.ProductID,
		CustomerID:  req.CustomerID,
		Description: desc,
		Rating:      req.Rating,
		Status:      domain.ReviewStatusPending, // reviews start as pending for moderation
		CreatedAt:   time.Now().UTC(),
	}

	if err := s.repo.SaveReview(ctx, review); err != nil {
		s.m.IncReviewAdded(metrics.StatusError)
		return domain.Review{}, err
	}

	s.m.IncReviewAdded(metrics.StatusSuccess)
	return review, nil
}

// GetReviews returns all approved reviews for a product.
func (s *Service) GetReviews(ctx context.Context, productID string) ([]domain.Review, error) {
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

// GetReviewSummary returns the review summary for a product.
func (s *Service) GetReviewSummary(ctx context.Context, productID string) (int64, int64, float64, error) {
	product, err := s.repo.GetByID(ctx, productID)
	if err != nil {
		return 0, 0, 0, err
	}
	avg := 0.0
	if product.TotalReviews > 0 {
		avg = float64(product.TotalRating) / float64(product.TotalReviews)
	}
	return product.TotalRating, product.TotalReviews, avg, nil
}

// GetRatingSummary returns the rating summary aggregate for a product.
func (s *Service) GetRatingSummary(ctx context.Context, productID string) (domain.RatingSummary, error) {
	return s.repo.GetRatingSummary(ctx, productID)
}
