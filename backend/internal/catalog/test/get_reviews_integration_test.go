//go:build integration

package test

import (
	"minimart/internal/catalog/application"
	"minimart/internal/catalog/domain"
	"minimart/internal/catalog/infrastructure/postgres"
	"minimart/pkg/outbox"
	outboxpg "minimart/pkg/outbox/postgres"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetReviewsIntegration(t *testing.T) {
	env := newCatalogEnv(t)

	t.Run("Product does not exist", func(t *testing.T) {
		_, err := env.Svc.GetReviews(t.Context(), "missing")
		require.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrProductNotFound)
	})

	t.Run("Approved reviews got returned", func(t *testing.T) {
		seedCatalogProduct(t, env, "p1", 3)
		now := time.Now().UTC()
		require.NoError(t, env.Repo.SaveReview(t.Context(), domain.Review{ID: "11111111-1111-4111-8111-111111111111", ProductID: "p1", CustomerID: "c1", Description: "Good", Rating: 5, Status: domain.ReviewStatusApproved, CreatedAt: now}))
		require.NoError(t, env.Repo.SaveReview(t.Context(), domain.Review{ID: "aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa", ProductID: "p1", CustomerID: "c2", Description: "Bad", Rating: 2, Status: domain.ReviewStatusApproved, CreatedAt: now.Add(time.Second)}))

		reviews, err := env.Svc.GetReviews(t.Context(), "p1")
		require.NoError(t, err)
		require.Len(t, reviews, 2)
		assert.Equal(t, "aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa", reviews[0].ID)
		assert.Equal(t, "11111111-1111-4111-8111-111111111111", reviews[1].ID)
	})

	t.Run("Pending reviews are not returned", func(t *testing.T) {
		seedCatalogProduct(t, env, "p1", 3)
		now := time.Now().UTC()
		require.NoError(t, env.Repo.SaveReview(t.Context(), domain.Review{ID: "11111111-1111-4111-8111-111111111111", ProductID: "p1", CustomerID: "c1", Description: "Good", Rating: 5, Status: domain.ReviewStatusApproved, CreatedAt: now}))
		require.NoError(t, env.Repo.SaveReview(t.Context(), domain.Review{ID: "bbbbbbbb-bbbb-4bbb-8bbb-bbbbbbbbbbbb", ProductID: "p1", CustomerID: "c3", Description: "Pending", Rating: 3, Status: domain.ReviewStatusPending, CreatedAt: now}))

		reviews, err := env.Svc.GetReviews(t.Context(), "p1")
		require.NoError(t, err)
		require.Len(t, reviews, 1)
		assert.Equal(t, "11111111-1111-4111-8111-111111111111", reviews[0].ID)
	})

	t.Run("The route is governed by the catalog:read permission", func(t *testing.T) {
		t.Skip("HTTP/authz-level assertion")
	})
	_ = application.AddReviewRequest{}
}

func TestCatalogRepository_GetReviewsReturnsOnlyApproved(t *testing.T) {
	pool, err := pgxpool.New(t.Context(), testDSN)
	require.NoError(t, err)
	defer pool.Close()

	repo := postgres.NewCatalogRepository(pool, pool, outbox.NewPublisher(outboxpg.NewStore(pool)))
	_, _ = pool.Exec(t.Context(), "DELETE FROM product_reviews WHERE product_id = $1", "it-review-product-2")
	_, _ = pool.Exec(t.Context(), "DELETE FROM product_reviews WHERE id IN ($1,$2)", "aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa", "bbbbbbbb-bbbb-4bbb-8bbb-bbbbbbbbbbbb")
	_, _ = pool.Exec(t.Context(), "DELETE FROM products WHERE id = $1", "it-review-product-2")
	seedReviewProduct(t, pool, repo, "it-review-product-2")

	approved := domain.Review{
		ID:          "aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa",
		ProductID:   "it-review-product-2",
		CustomerID:  "cust-it-2",
		Description: "Approved",
		Rating:      5,
		Status:      domain.ReviewStatusApproved,
		CreatedAt:   time.Now().UTC(),
	}
	pending := domain.Review{
		ID:          "bbbbbbbb-bbbb-4bbb-8bbb-bbbbbbbbbbbb",
		ProductID:   "it-review-product-2",
		CustomerID:  "cust-it-3",
		Description: "Pending",
		Rating:      3,
		Status:      domain.ReviewStatusPending,
		CreatedAt:   time.Now().UTC(),
	}
	require.NoError(t, repo.SaveReview(t.Context(), approved))
	require.NoError(t, repo.SaveReview(t.Context(), pending))

	reviews, err := repo.GetReviews(t.Context(), "it-review-product-2")
	require.NoError(t, err)
	require.Len(t, reviews, 1)
	assert.Equal(t, approved.ID, reviews[0].ID)
	assert.Equal(t, "Approved", reviews[0].Description)
}
