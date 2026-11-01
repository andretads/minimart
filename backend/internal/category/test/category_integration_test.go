//go:build integration

package test

import (
	"context"
	"minimart/internal/category/application"
	"minimart/internal/category/domain"
	"minimart/internal/category/infrastructure/postgres"
	"minimart/pkg/metrics"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCategoryIntegration(t *testing.T) {
	readPool, writePool := mustPools(t)
	repo := postgres.NewCategoryRepository(readPool, writePool)
	svc := application.NewService(repo, metrics.Noop{})

	t.Run("create and get category by ID", func(t *testing.T) {
		created, err := svc.CreateCategory(t.Context(), application.CreateCategoryRequest{
			Name: "Test1_Electronics",
			Icon: "📱",
		})
		require.NoError(t, err)
		assert.NotEmpty(t, created.ID)
		assert.Equal(t, "Test1_Electronics", created.Name)
		assert.Equal(t, "📱", created.Icon)
		assert.Equal(t, 1, created.Version)

		got, err := svc.GetCategory(t.Context(), created.ID)
		require.NoError(t, err)
		assert.Equal(t, created.ID, got.ID)
		assert.Equal(t, "Test1_Electronics", got.Name)
		assert.Equal(t, "📱", got.Icon)

		// Clean up to avoid polluting other tests
		err = svc.DeleteCategory(t.Context(), application.DeleteCategoryRequest{
			ID:      created.ID,
			Version: created.Version,
		})
		require.NoError(t, err)
	})

	t.Run("list categories ordered by name", func(t *testing.T) {
		_, err := svc.CreateCategory(t.Context(), application.CreateCategoryRequest{
			Name: "ListTest_Electronics",
			Icon: "📱",
		})
		require.NoError(t, err)
		_, err = svc.CreateCategory(t.Context(), application.CreateCategoryRequest{
			Name: "ListTest_Books",
			Icon: "📚",
		})
		require.NoError(t, err)
		_, err = svc.CreateCategory(t.Context(), application.CreateCategoryRequest{
			Name: "ListTest_Clothing",
			Icon: "👕",
		})
		require.NoError(t, err)

		got, err := svc.ListCategories(t.Context())
		require.NoError(t, err)
		require.Len(t, got, 3)
		assert.Equal(t, "ListTest_Books", got[0].Name)
		assert.Equal(t, "ListTest_Clothing", got[1].Name)
		assert.Equal(t, "ListTest_Electronics", got[2].Name)
	})

	t.Run("duplicate name returns ErrCategoryExists", func(t *testing.T) {
		_, err := svc.CreateCategory(t.Context(), application.CreateCategoryRequest{
			Name: "Test3_Electronics",
		})
		require.NoError(t, err)

		_, err = svc.CreateCategory(t.Context(), application.CreateCategoryRequest{
			Name: "test3_electronics",
		})
		require.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrCategoryExists)
	})

	t.Run("update category", func(t *testing.T) {
		created, err := svc.CreateCategory(t.Context(), application.CreateCategoryRequest{
			Name: "Test4_Electronics",
			Icon: "📱",
		})
		require.NoError(t, err)

		updated, err := svc.UpdateCategory(t.Context(), application.UpdateCategoryRequest{
			ID:      created.ID,
			Name:    "Test4_Gadgets",
			Icon:    "🔧",
			Version: created.Version,
		})
		require.NoError(t, err)
		assert.Equal(t, "Test4_Gadgets", updated.Name)
		assert.Equal(t, "🔧", updated.Icon)
		assert.Equal(t, 2, updated.Version)
		assert.WithinDuration(t, created.CreatedAt, updated.CreatedAt, 1*time.Second)

		got, err := svc.GetCategory(t.Context(), created.ID)
		require.NoError(t, err)
		assert.Equal(t, "Test4_Gadgets", got.Name)
		assert.Equal(t, "🔧", got.Icon)
	})

	t.Run("delete category", func(t *testing.T) {
		created, err := svc.CreateCategory(t.Context(), application.CreateCategoryRequest{
			Name: "Test5_Electronics",
			Icon: "📱",
		})
		require.NoError(t, err)

		err = svc.DeleteCategory(t.Context(), application.DeleteCategoryRequest{
			ID:      created.ID,
			Version: created.Version,
		})
		require.NoError(t, err)

		_, err = svc.GetCategory(t.Context(), created.ID)
		require.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrCategoryNotFound)
	})

	t.Run("get non-existent returns ErrCategoryNotFound", func(t *testing.T) {
		_, err := svc.GetCategory(t.Context(), "non-existent")
		require.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrCategoryNotFound)
	})

	t.Run("update with stale version returns ErrCategoryOptimisticLock", func(t *testing.T) {
		created, err := svc.CreateCategory(t.Context(), application.CreateCategoryRequest{
			Name: "Test6_Electronics",
		})
		require.NoError(t, err)

		// First update
		_, err = svc.UpdateCategory(t.Context(), application.UpdateCategoryRequest{
			ID:      created.ID,
			Name:    "Test6_Gadgets",
			Version: created.Version,
		})
		require.NoError(t, err)

		// Second update with stale version
		_, err = svc.UpdateCategory(t.Context(), application.UpdateCategoryRequest{
			ID:      created.ID,
			Name:    "Test6_Gadgets2",
			Version: created.Version,
		})
		require.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrCategoryOptimisticLock)
	})

	t.Run("delete with stale version returns ErrCategoryOptimisticLock", func(t *testing.T) {
		created, err := svc.CreateCategory(t.Context(), application.CreateCategoryRequest{
			Name: "Test7_Electronics",
		})
		require.NoError(t, err)

		err = svc.DeleteCategory(t.Context(), application.DeleteCategoryRequest{
			ID:      created.ID,
			Version: 99,
		})
		require.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrCategoryOptimisticLock)
	})
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
