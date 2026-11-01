//go:build integration

package test

import (
	"context"
	"minimart/internal/showcase/domain"
	"minimart/pkg/metrics"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"
)

type stubSigner struct{}

func (stubSigner) PresignGet(ctx context.Context, key string, expiry time.Duration) (string, error) {
	return "https://cdn.test/" + key, nil
}

func mustPools(t *testing.T) (*pgxpool.Pool, *pgxpool.Pool) {
	t.Helper()
	ctx := t.Context()
	readPool, err := newPool(ctx, testDSN)
	require.NoError(t, err)
	writePool, err := newPool(ctx, testDSN)
	require.NoError(t, err)
	return readPool, writePool
}

func newPool(ctx context.Context, dsn string) (*pgxpool.Pool, error) {
	cfg, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, err
	}
	return pgxpool.NewWithConfig(ctx, cfg)
}

func seedCategory(t *testing.T, pool *pgxpool.Pool, id, name string) {
	t.Helper()
	_, err := pool.Exec(t.Context(), `
		INSERT INTO categories (id, name, icon, version, created_at, updated_at)
		VALUES ($1, $2, '', 1, NOW(), NOW())
		ON CONFLICT (id) DO NOTHING
	`, id, name)
	require.NoError(t, err)
}

func seedProduct(t *testing.T, pool *pgxpool.Pool, p domain.Product, createdAt time.Time) {
	t.Helper()
	if createdAt.IsZero() {
		createdAt = time.Now().UTC()
	}
	_, err := pool.Exec(t.Context(), `
		INSERT INTO products (id, name, price_cents, stock, sku, description, version, created_at, updated_at, category_id)
		VALUES ($1, $2, $3, $4, NULLIF($5, ''), $6, 1, $7, $7, $8)
		ON CONFLICT (id) DO NOTHING
	`, p.ID, p.Name, p.PriceCents, p.Stock, p.SKU, p.Description, createdAt, p.CategoryID)
	require.NoError(t, err)
}

func seedImage(t *testing.T, pool *pgxpool.Pool, img domain.ProductImage) {
	t.Helper()
	_, err := pool.Exec(t.Context(), `
		INSERT INTO product_images (id, product_id, object_key, bucket, content_type, size_bytes, position, version, created_at, updated_at)
		VALUES ($1, $2, $3, 'bucket', 'image/png', 100, $4, 1, NOW(), NOW())
		ON CONFLICT (id) DO NOTHING
	`, img.ID, img.ProductID, img.ObjectKey, img.Position)
	require.NoError(t, err)
}

func seedReview(t *testing.T, pool *pgxpool.Pool, r domain.Review) {
	t.Helper()
	if r.CreatedAt.IsZero() {
		r.CreatedAt = time.Now().UTC()
	}
	_, err := pool.Exec(t.Context(), `
		INSERT INTO product_reviews (id, product_id, customer_id, description, rating, status, created_at)
		VALUES (CASE WHEN $1 = '' THEN gen_random_uuid() ELSE $1::uuid END, $2, $3, $4, $5, $6, $7)
		ON CONFLICT (product_id, customer_id) DO NOTHING
	`, r.ID, r.ProductID, r.CustomerID, r.Description, r.Rating, r.Status, r.CreatedAt)
	require.NoError(t, err)
}

var _ = metrics.Noop{}
