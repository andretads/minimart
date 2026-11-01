//go:build integration

package test

import (
	"encoding/json"
	"minimart/internal/catalog/domain"
	"minimart/internal/catalog/infrastructure/postgres"
	"minimart/pkg/message"
	"minimart/pkg/outbox"
	outboxpg "minimart/pkg/outbox/postgres"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// seedReviewProduct creates the category and product the reviews hang off.
func seedReviewProduct(t *testing.T, pool *pgxpool.Pool, repo domain.Repository, productID string) {
	t.Helper()

	_, err := pool.Exec(t.Context(), `
		INSERT INTO categories (id, name, icon, version, created_at, updated_at)
		VALUES ('cat-1', 'Test Category', '', 1, NOW(), NOW())
		ON CONFLICT (id) DO NOTHING
	`)
	require.NoError(t, err)

	p, err := domain.NewProduct(productID, "Integration Widget", 1299, 7, "cat-1", "", "", nil)
	require.NoError(t, err)
	require.NoError(t, repo.Save(t.Context(), p))
}

// TestCatalogRepository_SaveReviewWritesReviewAndOutboxEvent proves the review
// and its review-created event are written in the same transaction: the review
// row is readable and the outbox row carries the event payload.
func TestCatalogRepository_SaveReviewWritesReviewAndOutboxEvent(t *testing.T) {
	pool, err := pgxpool.New(t.Context(), testDSN)
	require.NoError(t, err)
	defer pool.Close()

	repo := postgres.NewCatalogRepository(pool, pool, outbox.NewPublisher(outboxpg.NewStore(pool)))
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
func TestCatalogRepository_GetReviewsReturnsOnlyApproved(t *testing.T) {
	pool, err := pgxpool.New(t.Context(), testDSN)
	require.NoError(t, err)
	defer pool.Close()

	repo := postgres.NewCatalogRepository(pool, pool, outbox.NewPublisher(outboxpg.NewStore(pool)))
	seedReviewProduct(t, pool, repo, "it-review-product-2")

	approved := domain.Review{
		ID:          "22222222-2222-4222-8222-222222222222",
		ProductID:   "it-review-product-2",
		CustomerID:  "cust-it-2",
		Description: "Approved",
		Rating:      5,
		Status:      domain.ReviewStatusApproved,
		CreatedAt:   time.Now().UTC(),
	}
	pending := domain.Review{
		ID:          "33333333-3333-4333-8333-333333333333",
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
