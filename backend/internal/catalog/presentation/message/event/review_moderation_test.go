//go:build unit

package event

import (
	"context"
	"encoding/json"
	"minimart/internal/catalog/application"
	"minimart/internal/catalog/domain"
	"minimart/internal/catalog/infrastructure/memory"
	"minimart/internal/catalog/testkit"
	"minimart/pkg/message"
	"minimart/pkg/metrics"
	"minimart/pkg/moderation"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// testModerationProvider is a test double for moderation.Provider.
type testModerationProvider struct {
	result moderation.Result
	err    error
}

func (f *testModerationProvider) Name() string { return "test" }

func (f *testModerationProvider) Moderate(ctx context.Context, content string) (moderation.Result, error) {
	if f.err != nil {
		return moderation.Result{}, f.err
	}
	return f.result, nil
}

// countingMetrics records review moderation results.
type countingMetrics struct {
	metrics.Noop
	moderated []string
}

func (c *countingMetrics) IncReviewModerated(decision string) {
	c.moderated = append(c.moderated, decision)
}

func TestHandler_ModerateReview_Approved(t *testing.T) {
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
	svc := application.NewService(repo, testkit.NewFakeS3Client(), fakeCategoryProvider{}, time.Hour, 5*1024*1024, m, provider)

	h := NewHandler(svc)

	msg, err := json.Marshal(message.ReviewCreated{ReviewID: "r1", CreatedAt: time.Now().UTC().Format(time.RFC3339Nano)})
	require.NoError(t, err)

	err = h.ModerateReview(t.Context(), msg)
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

func TestHandler_ModerateReview_Rejected(t *testing.T) {
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
	svc := application.NewService(repo, testkit.NewFakeS3Client(), fakeCategoryProvider{}, time.Hour, 5*1024*1024, m, provider)

	h := NewHandler(svc)

	msg, err := json.Marshal(message.ReviewCreated{ReviewID: "r1", CreatedAt: time.Now().UTC().Format(time.RFC3339Nano)})
	require.NoError(t, err)

	err = h.ModerateReview(t.Context(), msg)
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

func TestHandler_ModerateReview_NeedsReview(t *testing.T) {
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
	svc := application.NewService(repo, testkit.NewFakeS3Client(), fakeCategoryProvider{}, time.Hour, 5*1024*1024, m, provider)

	h := NewHandler(svc)

	msg, err := json.Marshal(message.ReviewCreated{ReviewID: "r1", CreatedAt: time.Now().UTC().Format(time.RFC3339Nano)})
	require.NoError(t, err)

	err = h.ModerateReview(t.Context(), msg)
	require.NoError(t, err)

	updated, err := repo.GetReviewByID(t.Context(), "r1")
	require.NoError(t, err)
	assert.Equal(t, domain.ReviewStatusNeedsReview, updated.Status)
	assert.Equal(t, "needs_review", updated.ModerationDecision)

	assert.Equal(t, []string{"needs_review"}, m.moderated)
}

func TestHandler_ModerateReview_AlreadyModerated(t *testing.T) {
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
	svc := application.NewService(repo, testkit.NewFakeS3Client(), fakeCategoryProvider{}, time.Hour, 5*1024*1024, m, provider)

	h := NewHandler(svc)

	msg, err := json.Marshal(message.ReviewCreated{ReviewID: "r1", CreatedAt: time.Now().UTC().Format(time.RFC3339Nano)})
	require.NoError(t, err)

	err = h.ModerateReview(t.Context(), msg)
	require.NoError(t, err)
}

func TestHandler_ModerateReview_ReviewNotFound(t *testing.T) {
	repo := memory.New()
	provider := &testModerationProvider{result: moderation.Result{Decision: "approved"}}
	m := &countingMetrics{}
	svc := application.NewService(repo, testkit.NewFakeS3Client(), fakeCategoryProvider{}, time.Hour, 5*1024*1024, m, provider)

	h := NewHandler(svc)

	msg, err := json.Marshal(message.ReviewCreated{ReviewID: "nonexistent", CreatedAt: time.Now().UTC().Format(time.RFC3339Nano)})
	require.NoError(t, err)

	err = h.ModerateReview(t.Context(), msg)
	require.ErrorIs(t, err, domain.ErrProductReviewNotFound)
	assert.Equal(t, []string{metrics.StatusError}, m.moderated)
}

func TestHandler_ModerateReview_InvalidJSON(t *testing.T) {
	repo := memory.New()
	provider := &testModerationProvider{result: moderation.Result{Decision: "approved"}}
	m := &countingMetrics{}
	svc := application.NewService(repo, testkit.NewFakeS3Client(), fakeCategoryProvider{}, time.Hour, 5*1024*1024, m, provider)

	h := NewHandler(svc)

	err := h.ModerateReview(t.Context(), []byte("not json"))
	require.Error(t, err)
}

func TestHandler_ModerateReview_ProviderError(t *testing.T) {
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
	svc := application.NewService(repo, testkit.NewFakeS3Client(), fakeCategoryProvider{}, time.Hour, 5*1024*1024, m, provider)

	h := NewHandler(svc)

	msg, err := json.Marshal(message.ReviewCreated{ReviewID: "r1", CreatedAt: time.Now().UTC().Format(time.RFC3339Nano)})
	require.NoError(t, err)

	err = h.ModerateReview(t.Context(), msg)
	require.ErrorIs(t, err, moderation.ErrProviderUnavailable)
	assert.Equal(t, []string{metrics.StatusError}, m.moderated)
}
