package application

import (
	"context"
	"minimart/internal/catalog/domain"
	"minimart/pkg/metrics"
	"minimart/pkg/moderation"
	"time"
)

type ModerationResult struct {
	Decision     string
	Confidence   float64
	Categories   []string
	Explanation  string
	Model        string
	ModelVersion string
	ModeratedAt  time.Time
}

type ModerateReviewRequest struct {
	ReviewID string
}

type ModerationService struct {
	repo     domain.Repository
	provider moderation.Provider
	metrics  metrics.Metrics
}

func NewModerationService(repo domain.Repository, provider moderation.Provider, m metrics.Metrics) *ModerationService {
	return &ModerationService{repo: repo, provider: provider, metrics: m}
}

func (s *ModerationService) ModerateReview(ctx context.Context, req ModerateReviewRequest) error {
	review, err := s.repo.GetReviewByID(ctx, req.ReviewID)
	if err != nil {
		s.metrics.IncReviewModerated(metrics.StatusError)
		return err
	}

	if review.Status != domain.ReviewStatusPending {
		return domain.ErrReviewAlreadyModerated
	}

	content := review.Description
	result, err := s.provider.Moderate(ctx, content)
	if err != nil {
		s.metrics.IncReviewModerated(metrics.StatusError)
		return err
	}

	// Validate result
	if err := s.validateResult(result); err != nil {
		s.metrics.IncReviewModerated(metrics.StatusError)
		return err
	}

	// Map moderation result to domain moderation result
	domainResult := moderation.Result{
		Decision:     result.Decision,
		Confidence:   result.Confidence,
		Categories:   result.Categories,
		Explanation:  result.Explanation,
		Model:        result.Model,
		ModelVersion: result.ModelVersion,
		ModeratedAt:  result.ModeratedAt,
	}

	if err := s.repo.UpdateReviewModeration(ctx, req.ReviewID, domainResult); err != nil {
		s.metrics.IncReviewModerated(metrics.StatusError)
		return err
	}

	// If approved, update product review counters
	if result.Decision == "approved" {
		s.updateProductCounters(ctx, review.ProductID)
	}

	s.metrics.IncReviewModerated(result.Decision)
	return nil
}

func (s *ModerationService) validateResult(result moderation.Result) error {
	validDecisions := map[string]bool{
		"approved":     true,
		"rejected":     true,
		"needs_review": true,
	}
	if !validDecisions[result.Decision] {
		return domain.ErrModerationFailed
	}
	if result.Confidence < 0 || result.Confidence > 1 {
		return domain.ErrModerationFailed
	}
	return nil
}

func (s *ModerationService) updateProductCounters(ctx context.Context, productID string) {
	reviews, err := s.repo.GetReviews(ctx, productID)
	if err != nil {
		s.metrics.IncReviewModerated(metrics.StatusError)
		return
	}

	var totalRating int64
	for _, r := range reviews {
		totalRating += int64(r.Rating)
	}
	totalReviews := len(reviews)

	if err := s.repo.UpdateReviewStats(ctx, productID, totalRating, int64(totalReviews)); err != nil {
		s.metrics.IncReviewModerated(metrics.StatusError)
	}
	if err := s.repo.UpdateRatingSummary(ctx, productID, totalRating, int64(totalReviews)); err != nil {
		s.metrics.IncReviewModerated(metrics.StatusError)
	}
}
