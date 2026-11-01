package application

import (
	"context"
	"errors"
	"minimart/internal/showcase/domain"
	"minimart/internal/showcase/infrastructure/memory"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestListCategories(t *testing.T) {
	t.Run("returns categories", func(t *testing.T) {
		repo := memory.New()
		repo.SaveCategory(domain.Category{ID: "c1", Name: "Books", Icon: "📚"})
		repo.SaveCategory(domain.Category{ID: "c2", Name: "Clothing", Icon: "👕"})
		svc := NewService(repo, nil, 0)

		got, err := svc.ListCategories(t.Context())
		require.NoError(t, err)
		require.Len(t, got, 2)
	})

	t.Run("returns empty when none", func(t *testing.T) {
		repo := memory.New()
		svc := NewService(repo, nil, 0)

		got, err := svc.ListCategories(t.Context())
		require.NoError(t, err)
		require.Empty(t, got)
	})

	t.Run("propagates repo error", func(t *testing.T) {
		repo := &errorCategoriesRepo{err: errors.New("db error")}
		svc := NewService(repo, nil, 0)

		_, err := svc.ListCategories(t.Context())
		require.Error(t, err)
		require.Contains(t, err.Error(), "db error")
	})
}

type errorCategoriesRepo struct {
	memory.Repository
	err error
}

func (r *errorCategoriesRepo) ListCategories(ctx context.Context) ([]domain.Category, error) {
	return nil, r.err
}
