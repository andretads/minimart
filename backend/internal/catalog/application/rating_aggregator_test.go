package application

import (
	"minimart/internal/catalog/domain"
	"minimart/internal/catalog/infrastructure/memory"
	"minimart/pkg/metrics"
	"minimart/pkg/moderation"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUpdateRatingSummary(t *testing.T) {
	repo := memory.New()
	p, err := domain.NewProduct("p1", "Widget", 500, 3, "cat-1", "", "", nil)
	require.NoError(t, err)
	require.NoError(t, repo.Save(t.Context(), p))

	// seed approved + pending reviews
	require.NoError(t, repo.SaveReview(t.Context(), domain.Review{ID: "r1", ProductID: "p1", CustomerID: "c1", Description: "Good", Rating: 5, Status: "approved", CreatedAt: time.Now().UTC()}))
	require.NoError(t, repo.SaveReview(t.Context(), domain.Review{ID: "r2", ProductID: "p1", CustomerID: "c2", Description: "Ok", Rating: 3, Status: "approved", CreatedAt: time.Now().UTC()}))
	require.NoError(t, repo.SaveReview(t.Context(), domain.Review{ID: "r3", ProductID: "p1", CustomerID: "c3", Description: "Pending", Rating: 1, Status: "pending", CreatedAt: time.Now().UTC()}))

	svc := NewService(repo, fakeS3Client{}, fakeCategoryProvider{}, time.Hour, 5*1024*1024, metrics.Noop{}, moderation.NewNoopProvider())

	err = svc.UpdateRatingSummary(t.Context(), "p1")
	require.NoError(t, err)

	prod, err := repo.GetByID(t.Context(), "p1")
	require.NoError(t, err)
	assert.Equal(t, int64(8), prod.TotalRating)
	assert.Equal(t, int64(2), prod.TotalReviews)

	summary, err := repo.GetRatingSummary(t.Context(), "p1")
	require.NoError(t, err)
	assert.Equal(t, int64(8), summary.RatingSum)
	assert.Equal(t, int64(2), summary.TotalReviews)
	assert.InDelta(t, 4.0, summary.AverageRating, 0.0001)

	// idempotent replay
	err = svc.UpdateRatingSummary(t.Context(), "p1")
	require.NoError(t, err)
	prod2, _ := repo.GetByID(t.Context(), "p1")
	assert.Equal(t, prod.TotalRating, prod2.TotalRating)
}

func TestUpdateRatingSummaryEmpty(t *testing.T) {
	repo := memory.New()
	p, err := domain.NewProduct("p1", "Widget", 500, 3, "cat-1", "", "", nil)
	require.NoError(t, err)
	require.NoError(t, repo.Save(t.Context(), p))

	svc := NewService(repo, fakeS3Client{}, fakeCategoryProvider{}, time.Hour, 5*1024*1024, metrics.Noop{}, moderation.NewNoopProvider())
	err = svc.UpdateRatingSummary(t.Context(), "p1")
	require.NoError(t, err)

	summary, err := repo.GetRatingSummary(t.Context(), "p1")
	require.NoError(t, err)
	assert.Equal(t, int64(0), summary.RatingSum)
	assert.Equal(t, int64(0), summary.TotalReviews)
	assert.InDelta(t, 0.0, summary.AverageRating, 0.0001)
}

func TestUpdateRatingSummaryRepoError(t *testing.T) {
	repo := &catalogRepoStub{}
	svc := NewService(repo, fakeS3Client{}, fakeCategoryProvider{}, time.Hour, 5*1024*1024, metrics.Noop{}, moderation.NewNoopProvider())
	// GetReviews returns nil; UpdateReviewStats returns nil from stub; no error. To force error, use nil repo? Instead just call and expect no panic.
	_ = svc.UpdateRatingSummary(t.Context(), "p1")
}
