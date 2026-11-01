package application

import (
	"context"
)

// UpdateRatingSummary recomputes the rating aggregate for a product from its
// approved reviews. It is idempotent and safe to replay.
func (s *Service) UpdateRatingSummary(ctx context.Context, productID string) error {
	reviews, err := s.repo.GetReviews(ctx, productID)
	if err != nil {
		return err
	}
	var sum int64
	for _, r := range reviews {
		sum += int64(r.Rating)
	}
	total := len(reviews)
	if err := s.repo.UpdateReviewStats(ctx, productID, sum, int64(total)); err != nil {
		return err
	}
	if err := s.repo.UpdateRatingSummary(ctx, productID, sum, int64(total)); err != nil {
		return err
	}
	return nil
}
