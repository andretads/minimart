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

func TestGetReviewSummary(t *testing.T) {
	t.Run("Product does not exist", func(t *testing.T) {
		svc, _ := newTestService(t)

		_, _, _, err := svc.GetReviewSummary(t.Context(), "missing")
		require.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrProductNotFound)
	})

	t.Run("Review summary got returned", func(t *testing.T) {
		repo := memory.New()
		p, err := domain.NewProduct("p1", "Widget", 500, 3, "cat-1", "", "", nil)
		require.NoError(t, err)
		p.TotalRating = 20
		p.TotalReviews = 5
		require.NoError(t, repo.Save(t.Context(), p))

		svc := NewService(repo, fakeS3Client{}, fakeCategoryProvider{}, time.Hour, 5*1024*1024, metrics.Noop{}, moderation.NewNoopProvider())

		totalRating, totalReviews, avg, err := svc.GetReviewSummary(t.Context(), "p1")
		require.NoError(t, err)
		assert.Equal(t, int64(20), totalRating)
		assert.Equal(t, int64(5), totalReviews)
		assert.InDelta(t, 4.0, avg, 0.0001)
	})
}
