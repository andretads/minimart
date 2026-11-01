//go:build integration

package test

import (
	"context"
	"minimart/internal/catalog/application"
	"minimart/internal/catalog/domain"
	"minimart/internal/catalog/infrastructure/postgres"
	"minimart/internal/catalog/testkit"
	"minimart/internal/category/presentation/module/client"
	"minimart/pkg/metrics"
	"minimart/pkg/moderation"
	"minimart/pkg/outbox"
	outboxpg "minimart/pkg/outbox/postgres"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"
)

type catalogTestEnv struct {
	Pool *pgxpool.Pool
	Repo *postgres.CatalogRepository
	Svc  *application.Service
}

func newCatalogEnv(t *testing.T) *catalogTestEnv {
	t.Helper()
	pool, err := pgxpool.New(t.Context(), testDSN)
	require.NoError(t, err)
	t.Cleanup(pool.Close)

	require.NoError(t, seedTestCategory(t.Context(), pool))

	_, _ = pool.Exec(t.Context(), "DELETE FROM stock_reservations")
	_, _ = pool.Exec(t.Context(), "DELETE FROM product_images")
	_, _ = pool.Exec(t.Context(), "DELETE FROM product_reviews")
	_, _ = pool.Exec(t.Context(), "DELETE FROM products")

	repo := postgres.NewCatalogRepository(pool, pool, outbox.NewPublisher(outboxpg.NewStore(pool)))
	svc := application.NewService(repo, testkit.NewFakeS3Client(), &fakeCategoryProvider{}, time.Hour, 5*1024*1024, metrics.Noop{}, moderation.NewNoopProvider())
	return &catalogTestEnv{Pool: pool, Repo: repo, Svc: svc}
}

func seedCatalogProduct(t *testing.T, env *catalogTestEnv, id string, stock int) {
	t.Helper()
	_, _ = env.Pool.Exec(t.Context(), "DELETE FROM stock_reservations WHERE product_id = $1", id)
	_, _ = env.Pool.Exec(t.Context(), "DELETE FROM product_reviews WHERE product_id = $1", id)
	_, _ = env.Pool.Exec(t.Context(), "DELETE FROM product_images WHERE product_id = $1", id)
	_, _ = env.Pool.Exec(t.Context(), "DELETE FROM products WHERE id = $1", id)
	p, err := domain.NewProduct(id, "Widget", 500, stock, "cat-1", "", "", nil)
	require.NoError(t, err)
	require.NoError(t, env.Repo.Save(t.Context(), p))
}

func seedCatalogProductFull(t *testing.T, env *catalogTestEnv, id string, name string, price int64, stock int, sku, desc string, details []domain.ProductDetail) {
	t.Helper()
	_, _ = env.Pool.Exec(t.Context(), "DELETE FROM stock_reservations WHERE product_id = $1", id)
	_, _ = env.Pool.Exec(t.Context(), "DELETE FROM product_reviews WHERE product_id = $1", id)
	_, _ = env.Pool.Exec(t.Context(), "DELETE FROM product_images WHERE product_id = $1", id)
	_, _ = env.Pool.Exec(t.Context(), "DELETE FROM products WHERE id = $1", id)
	p, err := domain.NewProduct(id, name, price, stock, "cat-1", sku, desc, details)
	require.NoError(t, err)
	require.NoError(t, env.Repo.Save(t.Context(), p))
}

type integrationModerationProvider struct {
	decision string
}

func (p *integrationModerationProvider) Name() string { return "integration" }

func (p *integrationModerationProvider) Moderate(ctx context.Context, content string) (moderation.Result, error) {
	return moderation.Result{
		Decision:     p.decision,
		Confidence:   0.9,
		Categories:   []string{},
		Explanation:  "test",
		Model:        "test",
		ModelVersion: "1",
		ModeratedAt:  time.Now().UTC(),
	}, nil
}

func seedTestCategory(ctx context.Context, pool *pgxpool.Pool) error {
	_, err := pool.Exec(ctx, `
		INSERT INTO categories (id, name, icon, version, created_at, updated_at)
		VALUES ('cat-1', 'Test Category', '', 1, NOW(), NOW())
		ON CONFLICT (id) DO NOTHING
	`)
	return err
}

// fakeCategoryProvider is a test double for CategoryProvider.
type fakeCategoryProvider struct{}

func (f *fakeCategoryProvider) GetCategory(ctx context.Context, id string) (client.CategoryDTO, error) {
	return client.CategoryDTO{ID: id, Name: "Test Category"}, nil
}

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
