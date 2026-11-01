package application

import (
	"context"
	"errors"
	"minimart/internal/catalog/domain"
	"minimart/internal/catalog/infrastructure/memory"
	"minimart/pkg/metrics"
	"minimart/pkg/moderation"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type getProductErrRepo struct {
	catalogRepoStub
	err error
}

func (f getProductErrRepo) GetByID(context.Context, string) (domain.Product, error) {
	return domain.Product{}, f.err
}

func TestGetProduct(t *testing.T) {
	t.Run("Product does not exist", func(t *testing.T) {
		svc := NewService(memory.New(), fakeS3Client{}, fakeCategoryProvider{}, time.Hour, 5*1024*1024, metrics.Noop{}, moderation.NewNoopProvider())

		_, err := svc.GetProduct(t.Context(), "missing")
		require.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrProductNotFound)
	})

	t.Run("Product got returned with no problems", func(t *testing.T) {
		repo := memory.New()
		seedProduct(t, repo, "p1", 3)
		svc := NewService(repo, fakeS3Client{}, fakeCategoryProvider{}, time.Hour, 5*1024*1024, metrics.Noop{}, moderation.NewNoopProvider())

		got, err := svc.GetProduct(t.Context(), "p1")
		require.NoError(t, err)
		assert.Equal(t, "p1", got.ID)
		assert.Equal(t, "Widget", got.Name)
		assert.Equal(t, int64(500), got.PriceCents)
		assert.Equal(t, 3, got.Stock)
		assert.Equal(t, 1, got.Version)
		assert.False(t, got.CreatedAt.IsZero())
	})

	t.Run("Product with description returned", func(t *testing.T) {
		repo := memory.New()
		p, err := domain.NewProduct("p1", "Widget", 500, 3, "cat-1", "", "A great widget", nil)
		require.NoError(t, err)
		require.NoError(t, repo.Save(t.Context(), p))
		svc := NewService(repo, fakeS3Client{}, fakeCategoryProvider{}, time.Hour, 5*1024*1024, metrics.Noop{}, moderation.NewNoopProvider())

		got, err := svc.GetProduct(t.Context(), "p1")
		require.NoError(t, err)
		assert.Equal(t, "A great widget", got.Description)
	})

	t.Run("Product with SKU returned", func(t *testing.T) {
		repo := memory.New()
		p, err := domain.NewProduct("p1", "Widget", 500, 3, "cat-1", "SKU-123", "", nil)
		require.NoError(t, err)
		require.NoError(t, repo.Save(t.Context(), p))
		svc := NewService(repo, fakeS3Client{}, fakeCategoryProvider{}, time.Hour, 5*1024*1024, metrics.Noop{}, moderation.NewNoopProvider())

		got, err := svc.GetProduct(t.Context(), "p1")
		require.NoError(t, err)
		assert.Equal(t, "SKU-123", got.SKU)
	})

	t.Run("Product with details returned", func(t *testing.T) {
		repo := memory.New()
		p, err := domain.NewProduct("p1", "Widget", 500, 3, "cat-1", "", "", []domain.ProductDetail{{Name: "Color", Value: "Red"}})
		require.NoError(t, err)
		require.NoError(t, repo.Save(t.Context(), p))
		svc := NewService(repo, fakeS3Client{}, fakeCategoryProvider{}, time.Hour, 5*1024*1024, metrics.Noop{}, moderation.NewNoopProvider())

		got, err := svc.GetProduct(t.Context(), "p1")
		require.NoError(t, err)
		assert.Equal(t, []domain.ProductDetail{{Name: "Color", Value: "Red"}}, got.Details)
	})

	t.Run("Unchanged product answers 304", func(t *testing.T) {
		repo := memory.New()
		seedProduct(t, repo, "p1", 3)
		svc := NewService(repo, fakeS3Client{}, fakeCategoryProvider{}, time.Hour, 5*1024*1024, metrics.Noop{}, moderation.NewNoopProvider())

		got, err := svc.GetProduct(t.Context(), "p1")
		require.NoError(t, err)
		assert.Equal(t, 1, got.Version)
	})

	t.Run("passes repository errors through unchanged", func(t *testing.T) {
		boom := errors.New("boom")
		svc := NewService(getProductErrRepo{err: boom}, fakeS3Client{}, fakeCategoryProvider{}, time.Hour, 5*1024*1024, metrics.Noop{}, moderation.NewNoopProvider())

		_, err := svc.GetProduct(t.Context(), "p1")
		assert.ErrorIs(t, err, boom)
	})
}
