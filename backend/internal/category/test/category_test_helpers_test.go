//go:build integration

package test

import (
	"context"
	"minimart/internal/category/application"
	"minimart/internal/category/domain"
	"minimart/internal/category/infrastructure/postgres"
	"minimart/pkg/metrics"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"
)

func newTestService(t *testing.T) *application.Service {
	t.Helper()
	readPool, writePool := mustPools(t)
	repo := postgres.NewCategoryRepository(readPool, writePool)
	return application.NewService(repo, metrics.Noop{})
}

func validCreateCategoryRequest() application.CreateCategoryRequest {
	return application.CreateCategoryRequest{
		Name: "Electronics",
		Icon: "📱",
	}
}

func validUpdateCategoryRequest(id string, version int) application.UpdateCategoryRequest {
	return application.UpdateCategoryRequest{
		ID:      id,
		Name:    "Gadgets",
		Icon:    "🔧",
		Version: version,
	}
}

func createTestCategory(t *testing.T, svc *application.Service, req application.CreateCategoryRequest) domain.Category {
	t.Helper()
	c, err := svc.CreateCategory(t.Context(), req)
	require.NoError(t, err)
	t.Cleanup(func() {
		ctx := context.WithoutCancel(t.Context())
		current, err := svc.GetCategory(ctx, c.ID)
		if err != nil {
			return
		}
		_ = svc.DeleteCategory(ctx, application.DeleteCategoryRequest{
			ID:      c.ID,
			Version: current.Version,
		})
	})
	return c
}

func deleteAllCategories(t *testing.T, svc *application.Service) {
	t.Helper()
	categories, err := svc.ListCategories(t.Context())
	require.NoError(t, err)
	for _, c := range categories {
		err := svc.DeleteCategory(t.Context(), application.DeleteCategoryRequest{
			ID:      c.ID,
			Version: c.Version,
		})
		require.NoError(t, err)
	}
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
