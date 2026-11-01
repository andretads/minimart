package module

import (
	"context"
	"errors"
	"minimart/internal/category/application"
	"minimart/internal/category/domain"
	"minimart/internal/category/infrastructure/memory"
	"minimart/internal/category/presentation/module/client"
	"minimart/pkg/metrics"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type failingRepo struct {
	err error
}

func (f failingRepo) GetByID(context.Context, string) (domain.Category, error) {
	return domain.Category{}, f.err
}

func (f failingRepo) List(context.Context) ([]domain.Category, error) { return nil, nil }
func (f failingRepo) Save(context.Context, domain.Category) error     { return nil }
func (f failingRepo) Update(context.Context, domain.Category) (domain.Category, error) {
	return domain.Category{}, nil
}
func (f failingRepo) Delete(context.Context, string, int) error { return nil }

func TestGetCategory(t *testing.T) {
	cat, err := domain.NewCategory("c1", "Electronics", "📱")
	require.NoError(t, err)

	t.Run("maps a domain category into the client type", func(t *testing.T) {
		repo := memory.New()
		require.NoError(t, repo.Save(t.Context(), cat))
		adapter := New(application.NewService(repo, metrics.Noop{}))

		got, err := adapter.GetCategory(t.Context(), "c1")
		require.NoError(t, err)
		assert.Equal(t, client.CategoryDTO{ID: "c1", Name: "Electronics", Icon: "📱"}, got)
	})

	t.Run("translates not-found into the client sentinel", func(t *testing.T) {
		adapter := New(application.NewService(memory.New(), metrics.Noop{}))

		_, err := adapter.GetCategory(t.Context(), "missing")
		assert.ErrorIs(t, err, client.ErrCategoryNotFound)
	})

	t.Run("passes through unexpected errors unchanged", func(t *testing.T) {
		boom := errors.New("boom")
		adapter := New(application.NewService(failingRepo{err: boom}, metrics.Noop{}))

		_, err := adapter.GetCategory(t.Context(), "c1")
		require.ErrorIs(t, err, boom)
		assert.NotErrorIs(t, err, client.ErrCategoryNotFound)
	})
}
