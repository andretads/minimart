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

func TestGetReviews(t *testing.T) {
	t.Run("Product does not exist", func(t *testing.T) {
		svc, _ := newTestService(t)

		_, err := svc.GetReviews(t.Context(), "missing")
		require.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrProductNotFound)
	})

	t.Run("Approved reviews got returned", func(t *testing.T) {
		repo := memory.New()
		p, err := domain.NewProduct("p1", "Widget", 500, 3, "cat-1", "", "", nil)
		require.NoError(t, err)
		require.NoError(t, repo.Save(t.Context(), p))

		now := time.Now().UTC()
		require.NoError(t, repo.SaveReview(t.Context(), domain.Review{ID: "r1", ProductID: "p1", CustomerID: "c1", Description: "Good", Rating: 5, Status: domain.ReviewStatusApproved, CreatedAt: now}))
		require.NoError(t, repo.SaveReview(t.Context(), domain.Review{ID: "r2", ProductID: "p1", CustomerID: "c2", Description: "Bad", Rating: 2, Status: domain.ReviewStatusApproved, CreatedAt: now.Add(time.Second)}))

		svc := NewService(repo, fakeS3Client{}, fakeCategoryProvider{}, time.Hour, 5*1024*1024, metrics.Noop{}, moderation.NewNoopProvider())

		reviews, err := svc.GetReviews(t.Context(), "p1")
		require.NoError(t, err)
		require.Len(t, reviews, 2)
		assert.Equal(t, "r2", reviews[0].ID)
		assert.Equal(t, "r1", reviews[1].ID)
	})

	t.Run("Pending reviews are not returned", func(t *testing.T) {
		repo := memory.New()
		p, err := domain.NewProduct("p1", "Widget", 500, 3, "cat-1", "", "", nil)
		require.NoError(t, err)
		require.NoError(t, repo.Save(t.Context(), p))

		now := time.Now().UTC()
		require.NoError(t, repo.SaveReview(t.Context(), domain.Review{ID: "r1", ProductID: "p1", CustomerID: "c1", Description: "Good", Rating: 5, Status: domain.ReviewStatusApproved, CreatedAt: now}))
		require.NoError(t, repo.SaveReview(t.Context(), domain.Review{ID: "r3", ProductID: "p1", CustomerID: "c3", Description: "Pending", Rating: 3, Status: domain.ReviewStatusPending, CreatedAt: now}))

		svc := NewService(repo, fakeS3Client{}, fakeCategoryProvider{}, time.Hour, 5*1024*1024, metrics.Noop{}, moderation.NewNoopProvider())

		reviews, err := svc.GetReviews(t.Context(), "p1")
		require.NoError(t, err)
		require.Len(t, reviews, 1)
		assert.Equal(t, "r1", reviews[0].ID)
	})

	t.Run("The route is governed by the catalog:read permission", func(t *testing.T) {
		t.Skip("HTTP/authz-level assertion")
	})
}
