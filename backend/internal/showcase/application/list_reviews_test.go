package application

import (
	"context"
	"errors"
	"minimart/internal/showcase/domain"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestListReviews(t *testing.T) {
	reviews := []domain.Review{
		{ID: "r1", ProductID: "p1", CustomerID: "c1", Description: "Great", Rating: 5, Status: "approved", CreatedAt: time.Now()},
	}

	t.Run("returns reviews for product", func(t *testing.T) {
		repo := &fakeRepo{reviews: map[string][]domain.Review{"p1": reviews}}
		svc := NewService(repo, nil, 0)

		got, err := svc.ListReviews(t.Context(), ListReviewsRequest{ProductID: "p1", Page: 1, PageSize: 20})
		require.NoError(t, err)
		require.Equal(t, reviews, got.Items)
	})

	t.Run("returns empty for product without reviews", func(t *testing.T) {
		repo := &fakeRepo{reviews: map[string][]domain.Review{}}
		svc := NewService(repo, nil, 0)

		got, err := svc.ListReviews(t.Context(), ListReviewsRequest{ProductID: "p1", Page: 1, PageSize: 20})
		require.NoError(t, err)
		require.Empty(t, got.Items)
	})

	t.Run("propagates repo error", func(t *testing.T) {
		repo := &errorReviewsRepo{err: errors.New("db error")}
		svc := NewService(repo, nil, 0)

		_, err := svc.ListReviews(t.Context(), ListReviewsRequest{ProductID: "p1", Page: 1, PageSize: 20})
		require.Error(t, err)
		require.Contains(t, err.Error(), "db error")
	})
}

type errorReviewsRepo struct {
	fakeRepo
	err error
}

func (r *errorReviewsRepo) ListReviews(ctx context.Context, productID string, limit, offset int) ([]domain.Review, error) {
	return nil, r.err
}
