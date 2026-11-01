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
