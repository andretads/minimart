package application

import (
	"context"
	"minimart/internal/catalog/domain"
)

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
