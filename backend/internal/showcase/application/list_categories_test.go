package application

import (
	"context"
	"errors"
	"minimart/internal/showcase/domain"
	"testing"

	"github.com/stretchr/testify/require"
)

type listCategoriesFakeRepo struct {
	fakeRepo
	categories []domain.Category
	err        error
}

func (r *listCategoriesFakeRepo) ListCategories(ctx context.Context) ([]domain.Category, error) {
	if r.err != nil {
		return nil, r.err
	}
	return r.categories, nil
}

func TestListCategories(t *testing.T) {
	categories := []domain.Category{
		{ID: "c1", Name: "Books", Icon: "📚"},
		{ID: "c2", Name: "Clothing", Icon: "👕"},
		{ID: "c3", Name: "Electronics", Icon: "📱"},
	}

	t.Run("returns categories", func(t *testing.T) {
		repo := &listCategoriesFakeRepo{categories: categories}
		svc := NewService(repo, nil, 0)

		got, err := svc.ListCategories(t.Context())
		require.NoError(t, err)
		require.Equal(t, categories, got)
	})

	t.Run("returns empty when none", func(t *testing.T) {
		repo := &listCategoriesFakeRepo{categories: []domain.Category{}}
		svc := NewService(repo, nil, 0)

		got, err := svc.ListCategories(t.Context())
		require.NoError(t, err)
		require.Empty(t, got)
	})

	t.Run("propagates repo error", func(t *testing.T) {
		repo := &listCategoriesFakeRepo{err: errors.New("db error")}
		svc := NewService(repo, nil, 0)

		_, err := svc.ListCategories(t.Context())
		require.Error(t, err)
		require.Contains(t, err.Error(), "db error")
	})
}
