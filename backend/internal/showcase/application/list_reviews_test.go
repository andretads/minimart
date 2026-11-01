package application

import (
	"context"
	"errors"
	"minimart/internal/showcase/domain"
	"minimart/internal/showcase/infrastructure/memory"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestListReviews(t *testing.T) {
	t.Run("Product with approved reviews returns them newest first", func(t *testing.T) {
		svc, repo := newTestService(t)

		now := time.Now().UTC()
		seedReview(t, repo, domain.Review{ID: "r1", ProductID: "p1", CustomerID: "c1", Description: "Older", Rating: 3, Status: "approved", CreatedAt: now.Add(-time.Hour)})
		seedReview(t, repo, domain.Review{ID: "r2", ProductID: "p1", CustomerID: "c2", Description: "Newer", Rating: 5, Status: "approved", CreatedAt: now})

		result, err := svc.ListReviews(t.Context(), ListReviewsRequest{ProductID: "p1", Page: 1, PageSize: 20})
		require.NoError(t, err)
		require.Len(t, result.Items, 2)
		assert.Equal(t, "r2", result.Items[0].ID)
		assert.Equal(t, "r1", result.Items[1].ID)
		for _, r := range result.Items {
			assert.NotEmpty(t, r.ID)
			assert.Equal(t, "p1", r.ProductID)
			assert.NotEmpty(t, r.CustomerID)
			assert.NotEmpty(t, r.Description)
			assert.NotZero(t, r.Rating)
			assert.Equal(t, "approved", r.Status)
			assert.False(t, r.CreatedAt.IsZero())
		}
	})

	t.Run("Pending reviews are excluded", func(t *testing.T) {
		svc, repo := newTestService(t)

		now := time.Now().UTC()
		seedReview(t, repo, domain.Review{ID: "r1", ProductID: "p1", CustomerID: "c1", Description: "Approved", Rating: 5, Status: "approved", CreatedAt: now})
		seedReview(t, repo, domain.Review{ID: "r2", ProductID: "p1", CustomerID: "c2", Description: "Pending", Rating: 3, Status: "pending", CreatedAt: now})

		result, err := svc.ListReviews(t.Context(), ListReviewsRequest{ProductID: "p1", Page: 1, PageSize: 20})
		require.NoError(t, err)
		require.Len(t, result.Items, 1)
		assert.Equal(t, "c1", result.Items[0].CustomerID)
	})

	t.Run("Product without reviews returns an empty array", func(t *testing.T) {
		svc, _ := newTestService(t)

		result, err := svc.ListReviews(t.Context(), ListReviewsRequest{ProductID: "p1", Page: 1, PageSize: 20})
		require.NoError(t, err)
		assert.Empty(t, result.Items)
	})

	t.Run("Repository failure returns 500", func(t *testing.T) {
		repo := &errorReviewsRepo{err: errors.New("db error")}
		svc := NewService(repo, nil, 0)

		_, err := svc.ListReviews(t.Context(), ListReviewsRequest{ProductID: "p1", Page: 1, PageSize: 20})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "db error")
	})
}

type errorReviewsRepo struct {
	memory.Repository
	err error
}

func (r *errorReviewsRepo) ListReviews(ctx context.Context, productID string, limit, offset int) ([]domain.Review, error) {
	return nil, r.err
}

func (r *errorReviewsRepo) CountReviews(ctx context.Context, productID string) (int, error) {
	return 0, r.err
}
