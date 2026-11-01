package application

import (
	"context"
	"minimart/internal/catalog/domain"
	"minimart/internal/catalog/infrastructure/memory"
	"minimart/pkg/metrics"
	"minimart/pkg/moderation"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// testModerationProvider is a configurable test double for moderation.Provider.
type testModerationProvider struct {
	result moderation.Result
	err    error
}

func (f *testModerationProvider) Name() string {
	return "test"
}

func (f *testModerationProvider) Moderate(ctx context.Context, content string) (moderation.Result, error) {
	if f.err != nil {
		return moderation.Result{}, f.err
	}
	return f.result, nil
}

func TestModerateReview_Approved(t *testing.T) {
	repo := memory.New()
	p, err := domain.NewProduct("p1", "Widget", 500, 3, "cat-1", "", "", nil)
	require.NoError(t, err)
	require.NoError(t, repo.Save(t.Context(), p))

	review := domain.Review{
		ID:          "r1",
		ProductID:   "p1",
		CustomerID:  "c1",
		Description: "Great product!",
		Rating:      5,
		Status:      domain.ReviewStatusPending,
		CreatedAt:   time.Now().UTC(),
	}
	require.NoError(t, repo.SaveReview(t.Context(), review))

	provider := &testModerationProvider{
		result: moderation.Result{
			Decision:     "approved",
			Confidence:   0.95,
			Categories:   []string{},
			Explanation:  "Content is safe",
			Model:        "llama-guard3",
			ModelVersion: "1b",
			ModeratedAt:  time.Now().UTC(),
		},
	}
	m := &countingMetrics{}
	svc := NewModerationService(repo, provider, m)

	err = svc.ModerateReview(t.Context(), ModerateReviewRequest{ReviewID: "r1"})
	require.NoError(t, err)

	updated, err := repo.GetReviewByID(t.Context(), "r1")
	require.NoError(t, err)
	assert.Equal(t, domain.ReviewStatusApproved, updated.Status)
	assert.Equal(t, "approved", updated.ModerationDecision)
	assert.InEpsilon(t, 0.95, updated.ModerationConfidence, 0.001)
	assert.NotNil(t, updated.ModeratedAt)

	product, err := repo.GetByID(t.Context(), "p1")
	require.NoError(t, err)
	assert.Equal(t, int64(5), product.TotalRating)
	assert.Equal(t, int64(1), product.TotalReviews)

	assert.Equal(t, []string{"approved"}, m.moderated)
}

func TestModerateReview_Rejected(t *testing.T) {
	repo := memory.New()
	p, err := domain.NewProduct("p1", "Widget", 500, 3, "cat-1", "", "", nil)
	require.NoError(t, err)
	require.NoError(t, repo.Save(t.Context(), p))

	review := domain.Review{
		ID:          "r1",
		ProductID:   "p1",
		CustomerID:  "c1",
		Description: "This is garbage!",
		Rating:      1,
		Status:      domain.ReviewStatusPending,
		CreatedAt:   time.Now().UTC(),
	}
	require.NoError(t, repo.SaveReview(t.Context(), review))

	provider := &testModerationProvider{
		result: moderation.Result{
			Decision:     "rejected",
			Confidence:   0.9,
			Categories:   []string{"profanity"},
			Explanation:  "Contains profanity",
			Model:        "llama-guard3",
			ModelVersion: "1b",
			ModeratedAt:  time.Now().UTC(),
		},
	}
	m := &countingMetrics{}
	svc := NewModerationService(repo, provider, m)

	err = svc.ModerateReview(t.Context(), ModerateReviewRequest{ReviewID: "r1"})
	require.NoError(t, err)

	updated, err := repo.GetReviewByID(t.Context(), "r1")
	require.NoError(t, err)
	assert.Equal(t, domain.ReviewStatusRejected, updated.Status)
	assert.Equal(t, "rejected", updated.ModerationDecision)
	assert.Contains(t, updated.ModerationCategories, "profanity")

	product, err := repo.GetByID(t.Context(), "p1")
	require.NoError(t, err)
	assert.Equal(t, int64(0), product.TotalRating)
	assert.Equal(t, int64(0), product.TotalReviews)

	assert.Equal(t, []string{"rejected"}, m.moderated)
}

func TestModerateReview_NeedsReview(t *testing.T) {
	repo := memory.New()
	p, err := domain.NewProduct("p1", "Widget", 500, 3, "cat-1", "", "", nil)
	require.NoError(t, err)
	require.NoError(t, repo.Save(t.Context(), p))

	review := domain.Review{
		ID:          "r1",
		ProductID:   "p1",
		CustomerID:  "c1",
		Description: "Hmm not sure",
		Rating:      3,
		Status:      domain.ReviewStatusPending,
		CreatedAt:   time.Now().UTC(),
	}
	require.NoError(t, repo.SaveReview(t.Context(), review))

	provider := &testModerationProvider{
		result: moderation.Result{
			Decision:     "needs_review",
			Confidence:   0.5,
			Categories:   []string{},
			Explanation:  "Ambiguous content",
			Model:        "llama-guard3",
			ModelVersion: "1b",
			ModeratedAt:  time.Now().UTC(),
		},
	}
	m := &countingMetrics{}
	svc := NewModerationService(repo, provider, m)

	err = svc.ModerateReview(t.Context(), ModerateReviewRequest{ReviewID: "r1"})
	require.NoError(t, err)

	updated, err := repo.GetReviewByID(t.Context(), "r1")
	require.NoError(t, err)
	assert.Equal(t, domain.ReviewStatusNeedsReview, updated.Status)
	assert.Equal(t, "needs_review", updated.ModerationDecision)

	assert.Equal(t, []string{"needs_review"}, m.moderated)
}

func TestModerateReview_AlreadyModerated(t *testing.T) {
	repo := memory.New()
	p, err := domain.NewProduct("p1", "Widget", 500, 3, "cat-1", "", "", nil)
	require.NoError(t, err)
	require.NoError(t, repo.Save(t.Context(), p))

	review := domain.Review{
		ID:          "r1",
		ProductID:   "p1",
		CustomerID:  "c1",
		Description: "Great!",
		Rating:      5,
		Status:      domain.ReviewStatusApproved,
		CreatedAt:   time.Now().UTC(),
	}
	require.NoError(t, repo.SaveReview(t.Context(), review))

	provider := &testModerationProvider{result: moderation.Result{Decision: "approved"}}
	m := &countingMetrics{}
	svc := NewModerationService(repo, provider, m)

	err = svc.ModerateReview(t.Context(), ModerateReviewRequest{ReviewID: "r1"})
	require.ErrorIs(t, err, domain.ErrReviewAlreadyModerated)
}

func TestModerateReview_ReviewNotFound(t *testing.T) {
	repo := memory.New()
	provider := &testModerationProvider{result: moderation.Result{Decision: "approved"}}
	m := &countingMetrics{}
	svc := NewModerationService(repo, provider, m)

	err := svc.ModerateReview(t.Context(), ModerateReviewRequest{ReviewID: "nonexistent"})
	require.ErrorIs(t, err, domain.ErrProductReviewNotFound)
	assert.Equal(t, []string{metrics.StatusError}, m.moderated)
}

func TestModerateReview_ProviderError(t *testing.T) {
	repo := memory.New()
	p, err := domain.NewProduct("p1", "Widget", 500, 3, "cat-1", "", "", nil)
	require.NoError(t, err)
	require.NoError(t, repo.Save(t.Context(), p))

	review := domain.Review{
		ID:          "r1",
		ProductID:   "p1",
		CustomerID:  "c1",
		Description: "Test",
		Rating:      5,
		Status:      domain.ReviewStatusPending,
		CreatedAt:   time.Now().UTC(),
	}
	require.NoError(t, repo.SaveReview(t.Context(), review))

	provider := &testModerationProvider{err: moderation.ErrProviderUnavailable}
	m := &countingMetrics{}
	svc := NewModerationService(repo, provider, m)

	err = svc.ModerateReview(t.Context(), ModerateReviewRequest{ReviewID: "r1"})
	require.ErrorIs(t, err, moderation.ErrProviderUnavailable)
	assert.Equal(t, []string{metrics.StatusError}, m.moderated)
}

func TestModerateReview_InvalidDecision(t *testing.T) {
	repo := memory.New()
	p, err := domain.NewProduct("p1", "Widget", 500, 3, "cat-1", "", "", nil)
	require.NoError(t, err)
	require.NoError(t, repo.Save(t.Context(), p))

	review := domain.Review{
		ID:          "r1",
		ProductID:   "p1",
		CustomerID:  "c1",
		Description: "Test",
		Rating:      5,
		Status:      domain.ReviewStatusPending,
		CreatedAt:   time.Now().UTC(),
	}
	require.NoError(t, repo.SaveReview(t.Context(), review))

	provider := &testModerationProvider{
		result: moderation.Result{
			Decision:   "invalid_decision",
			Confidence: 0.9,
		},
	}
	m := &countingMetrics{}
	svc := NewModerationService(repo, provider, m)

	err = svc.ModerateReview(t.Context(), ModerateReviewRequest{ReviewID: "r1"})
	require.ErrorIs(t, err, domain.ErrModerationFailed)
	assert.Equal(t, []string{metrics.StatusError}, m.moderated)
}

func TestModerateReview_InvalidConfidence(t *testing.T) {
	repo := memory.New()
	p, err := domain.NewProduct("p1", "Widget", 500, 3, "cat-1", "", "", nil)
	require.NoError(t, err)
	require.NoError(t, repo.Save(t.Context(), p))

	review := domain.Review{
		ID:          "r1",
		ProductID:   "p1",
		CustomerID:  "c1",
		Description: "Test",
		Rating:      5,
		Status:      domain.ReviewStatusPending,
		CreatedAt:   time.Now().UTC(),
	}
	require.NoError(t, repo.SaveReview(t.Context(), review))

	provider := &testModerationProvider{
		result: moderation.Result{
			Decision:   "approved",
			Confidence: 1.5,
		},
	}
	m := &countingMetrics{}
	svc := NewModerationService(repo, provider, m)

	err = svc.ModerateReview(t.Context(), ModerateReviewRequest{ReviewID: "r1"})
	require.ErrorIs(t, err, domain.ErrModerationFailed)
	assert.Equal(t, []string{metrics.StatusError}, m.moderated)
}

func TestModerateReview_ProviderTimeout(t *testing.T) {
	repo := memory.New()
	p, err := domain.NewProduct("p1", "Widget", 500, 3, "cat-1", "", "", nil)
	require.NoError(t, err)
	require.NoError(t, repo.Save(t.Context(), p))

	review := domain.Review{
		ID:          "r1",
		ProductID:   "p1",
		CustomerID:  "c1",
		Description: "Test",
		Rating:      5,
		Status:      domain.ReviewStatusPending,
		CreatedAt:   time.Now().UTC(),
	}
	require.NoError(t, repo.SaveReview(t.Context(), review))

	provider := &testModerationProvider{err: moderation.ErrTimeout}
	m := &countingMetrics{}
	svc := NewModerationService(repo, provider, m)

	err = svc.ModerateReview(t.Context(), ModerateReviewRequest{ReviewID: "r1"})
	require.ErrorIs(t, err, moderation.ErrTimeout)
	assert.Equal(t, []string{metrics.StatusError}, m.moderated)
}

func TestModerateReview_ValidatesCategoriesAndExplanation(t *testing.T) {
	repo := memory.New()
	p, err := domain.NewProduct("p1", "Widget", 500, 3, "cat-1", "", "", nil)
	require.NoError(t, err)
	require.NoError(t, repo.Save(t.Context(), p))

	review := domain.Review{
		ID:          "r1",
		ProductID:   "p1",
		CustomerID:  "c1",
		Description: "Bad content",
		Rating:      1,
		Status:      domain.ReviewStatusPending,
		CreatedAt:   time.Now().UTC(),
	}
	require.NoError(t, repo.SaveReview(t.Context(), review))

	provider := &testModerationProvider{
		result: moderation.Result{
			Decision:     "rejected",
			Confidence:   0.92,
			Categories:   []string{"hate_speech", "harassment"},
			Explanation:  "Contains hate speech and harassment",
			Model:        "llama-guard3",
			ModelVersion: "1b",
			ModeratedAt:  time.Now().UTC(),
		},
	}
	m := &countingMetrics{}
	svc := NewModerationService(repo, provider, m)

	err = svc.ModerateReview(t.Context(), ModerateReviewRequest{ReviewID: "r1"})
	require.NoError(t, err)

	updated, err := repo.GetReviewByID(t.Context(), "r1")
	require.NoError(t, err)
	assert.Equal(t, []string{"hate_speech", "harassment"}, updated.ModerationCategories)
	assert.Equal(t, "Contains hate speech and harassment", updated.ModerationExplanation)
	assert.Equal(t, "llama-guard3", updated.ModerationModel)
	assert.Equal(t, "1b", updated.ModerationModelVersion)
}
