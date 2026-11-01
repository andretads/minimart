//go:build integration

package test

import (
	"encoding/json"
	"minimart/internal/catalog/application"
	"minimart/internal/catalog/domain"
	"minimart/internal/catalog/infrastructure/postgres"
	"minimart/internal/catalog/testkit"
	"minimart/pkg/message"
	"minimart/pkg/metrics"
	"minimart/pkg/outbox"
	outboxpg "minimart/pkg/outbox/postgres"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAddReviewIntegration(t *testing.T) {
	env := newCatalogEnv(t)

	t.Run("Product does not exist", func(t *testing.T) {
		_, err := env.Svc.AddReview(t.Context(), application.AddReviewRequest{ProductID: "missing", CustomerID: "cust-1", Description: "Good", Rating: 4})
		require.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrProductNotFound)
	})

	t.Run("Invalid rating too low is rejected", func(t *testing.T) {
		seedCatalogProduct(t, env, "p1", 3)
		_, err := env.Svc.AddReview(t.Context(), application.AddReviewRequest{ProductID: "p1", CustomerID: "cust-1", Description: "Bad", Rating: 0})
		require.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrInvalidRating)
	})

	t.Run("Invalid rating too high is rejected", func(t *testing.T) {
		seedCatalogProduct(t, env, "p1", 3)
		_, err := env.Svc.AddReview(t.Context(), application.AddReviewRequest{ProductID: "p1", CustomerID: "cust-1", Description: "Bad", Rating: 6})
		require.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrInvalidRating)
	})

	t.Run("Empty description is rejected", func(t *testing.T) {
		seedCatalogProduct(t, env, "p1", 3)
		_, err := env.Svc.AddReview(t.Context(), application.AddReviewRequest{ProductID: "p1", CustomerID: "cust-1", Description: "   ", Rating: 5})
		require.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrEmptyName)
	})

	t.Run("Missing customer is rejected", func(t *testing.T) {
		seedCatalogProduct(t, env, "p1", 3)
		_, err := env.Svc.AddReview(t.Context(), application.AddReviewRequest{ProductID: "p1", CustomerID: "", Description: "Good", Rating: 4})
		require.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrEmptyCustomerID)
	})

	t.Run("Unauthenticated request is rejected", func(t *testing.T) {
		t.Skip("HTTP-level auth")
	})

	t.Run("Storage failure", func(t *testing.T) {
		t.Skip("storage failure not easily simulated with real DB")
	})

	t.Run("Review got added with no problems", func(t *testing.T) {
		seedCatalogProduct(t, env, "p1", 3)
		review, err := env.Svc.AddReview(t.Context(), application.AddReviewRequest{ProductID: "p1", CustomerID: "cust-1", Description: "Great product!", Rating: 5})
		require.NoError(t, err)
		assert.NotEmpty(t, review.ID)
		assert.Equal(t, "p1", review.ProductID)
		assert.Equal(t, "cust-1", review.CustomerID)
		assert.Equal(t, "Great product!", review.Description)
		assert.Equal(t, 5, review.Rating)
		assert.False(t, review.CreatedAt.IsZero())
	})

	t.Run("Review starts as pending for moderation", func(t *testing.T) {
		seedCatalogProduct(t, env, "p1", 3)
		review, err := env.Svc.AddReview(t.Context(), application.AddReviewRequest{ProductID: "p1", CustomerID: "cust-1", Description: "Great product!", Rating: 5})
		require.NoError(t, err)
		assert.Equal(t, domain.ReviewStatusPending, review.Status)
	})

	t.Run("Review is attributed to the authenticated customer", func(t *testing.T) {
		seedCatalogProduct(t, env, "p1", 3)
		review, err := env.Svc.AddReview(t.Context(), application.AddReviewRequest{ProductID: "p1", CustomerID: "cust-1", Description: "Great product!", Rating: 5})
		require.NoError(t, err)
		assert.Equal(t, "cust-1", review.CustomerID)
	})

	t.Run("Review is approved by moderation", func(t *testing.T) {
		seedCatalogProduct(t, env, "p1", 3)
		review, err := env.Svc.AddReview(t.Context(), application.AddReviewRequest{ProductID: "p1", CustomerID: "cust-1", Description: "Great product!", Rating: 5})
		require.NoError(t, err)

		provider := &integrationModerationProvider{decision: "approved"}
		svc := application.NewService(env.Repo, testkit.NewFakeS3Client(), &fakeCategoryProvider{}, time.Hour, 5*1024*1024, metrics.Noop{}, provider)
		err = svc.ModerateReview(t.Context(), application.ModerateReviewRequest{ReviewID: review.ID})
		require.NoError(t, err)

		updated, err := env.Repo.GetReviewByID(t.Context(), review.ID)
		require.NoError(t, err)
		assert.Equal(t, domain.ReviewStatusApproved, updated.Status)
		assert.NotEmpty(t, updated.ModerationDecision)
	})

	t.Run("Review is rejected by moderation", func(t *testing.T) {
		seedCatalogProduct(t, env, "p1", 3)
		review, err := env.Svc.AddReview(t.Context(), application.AddReviewRequest{ProductID: "p1", CustomerID: "cust-1", Description: "This product is f***ing garbage!", Rating: 1})
		require.NoError(t, err)

		provider := &integrationModerationProvider{decision: "rejected"}
		svc := application.NewService(env.Repo, testkit.NewFakeS3Client(), &fakeCategoryProvider{}, time.Hour, 5*1024*1024, metrics.Noop{}, provider)
		err = svc.ModerateReview(t.Context(), application.ModerateReviewRequest{ReviewID: review.ID})
		require.NoError(t, err)

		updated, err := env.Repo.GetReviewByID(t.Context(), review.ID)
		require.NoError(t, err)
		assert.Equal(t, domain.ReviewStatusRejected, updated.Status)

		reviews, err := env.Svc.GetReviews(t.Context(), "p1")
		require.NoError(t, err)
		for _, r := range reviews {
			assert.NotEqual(t, review.ID, r.ID)
		}
	})

	t.Run("Review needs human review", func(t *testing.T) {
		seedCatalogProduct(t, env, "p1", 3)
		review, err := env.Svc.AddReview(t.Context(), application.AddReviewRequest{ProductID: "p1", CustomerID: "cust-1", Description: "Hmm, not sure about this one", Rating: 3})
		require.NoError(t, err)

		provider := &integrationModerationProvider{decision: "needs_review"}
		svc := application.NewService(env.Repo, testkit.NewFakeS3Client(), &fakeCategoryProvider{}, time.Hour, 5*1024*1024, metrics.Noop{}, provider)
		err = svc.ModerateReview(t.Context(), application.ModerateReviewRequest{ReviewID: review.ID})
		require.NoError(t, err)

		updated, err := env.Repo.GetReviewByID(t.Context(), review.ID)
		require.NoError(t, err)
		assert.Equal(t, domain.ReviewStatusNeedsReview, updated.Status)

		reviews, err := env.Svc.GetReviews(t.Context(), "p1")
		require.NoError(t, err)
		for _, r := range reviews {
			assert.NotEqual(t, review.ID, r.ID)
		}
	})
}

func TestCatalogRepository_SaveReviewWritesReviewAndOutboxEvent(t *testing.T) {
	pool, err := pgxpool.New(t.Context(), testDSN)
	require.NoError(t, err)
	defer pool.Close()

	repo := postgres.NewCatalogRepository(pool, pool, outbox.NewPublisher(outboxpg.NewStore(pool)))
	_, _ = pool.Exec(t.Context(), "DELETE FROM product_reviews WHERE product_id = $1", "it-review-product")
	_, _ = pool.Exec(t.Context(), "DELETE FROM products WHERE id = $1", "it-review-product")
	seedReviewProduct(t, pool, repo, "it-review-product")

	review := domain.Review{
		ID:          "11111111-1111-4111-8111-111111111111",
		ProductID:   "it-review-product",
		CustomerID:  "cust-it-1",
		Description: "Great product!",
		Rating:      5,
		Status:      domain.ReviewStatusPending,
		CreatedAt:   time.Now().UTC(),
	}
	require.NoError(t, repo.SaveReview(t.Context(), review))

	var storedID, storedCustomer, storedStatus string
	err = pool.QueryRow(t.Context(), `
		SELECT id::text, customer_id, status FROM product_reviews WHERE id = $1
	`, review.ID).Scan(&storedID, &storedCustomer, &storedStatus)
	require.NoError(t, err)
	assert.Equal(t, review.ID, storedID)
	assert.Equal(t, "cust-it-1", storedCustomer)
	assert.Equal(t, domain.ReviewStatusPending, storedStatus)

	var topic, key string
	var body []byte
	err = pool.QueryRow(t.Context(), `
		SELECT topic, message_key, body FROM outbox WHERE message_key = $1
	`, review.ID).Scan(&topic, &key, &body)
	require.NoError(t, err)
	assert.Equal(t, "review-created", topic)
	assert.Equal(t, review.ID, key)

	var event message.ReviewCreated
	require.NoError(t, json.Unmarshal(body, &event))
	assert.Equal(t, review.ID, event.ReviewID)
}

// TestCatalogRepository_GetReviewsReturnsOnlyApproved proves the read path
// hides reviews that are still pending or rejected.
