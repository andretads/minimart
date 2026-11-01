package application

import (
	"context"
	"errors"
	"minimart/internal/catalog/domain"
	"minimart/internal/catalog/infrastructure/memory"
	"minimart/pkg/moderation"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type updateFailingRepo struct {
	catalogRepoStub
	product domain.Product
	err     error
}

func (f updateFailingRepo) GetByID(context.Context, string) (domain.Product, error) {
	if f.product.ID == "" {
		return domain.Product{}, domain.ErrProductNotFound
	}
	return f.product, nil
}

func (f updateFailingRepo) Update(context.Context, domain.Product) (domain.Product, error) {
	return domain.Product{}, f.err
}

func seedUpdateProduct(t *testing.T, repo *memory.Repository) {
	t.Helper()
	p, err := domain.NewProduct("p1", "Widget", 500, 3, "cat-1", "", "", nil)
	require.NoError(t, err)
	require.NoError(t, repo.Save(t.Context(), p))
}

func TestUpdateProduct(t *testing.T) {
	t.Run("Product does not exist", func(t *testing.T) {
		svc, _ := newTestService(t)

		_, err := svc.UpdateProduct(t.Context(), validUpdateProductRequest("missing", 1))
		require.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrProductNotFound)
	})

	t.Run("Blank name is rejected", func(t *testing.T) {
		svc, repo := newTestService(t)
		seedUpdateProduct(t, repo)

		req := validUpdateProductRequest("p1", 1)
		req.Name = "   "
		_, err := svc.UpdateProduct(t.Context(), req)
		require.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrEmptyName)
	})

	t.Run("Non-positive price is rejected", func(t *testing.T) {
		svc, repo := newTestService(t)
		seedUpdateProduct(t, repo)

		req := validUpdateProductRequest("p1", 1)
		req.PriceCents = 0
		_, err := svc.UpdateProduct(t.Context(), req)
		require.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrNonPositivePrice)
	})

	t.Run("Storage failure", func(t *testing.T) {
		existing, err := domain.NewProduct("p1", "Widget", 500, 3, "cat-1", "", "", nil)
		require.NoError(t, err)
		boom := errors.New("disk on fire")
		m := &countingMetrics{}
		svc := NewService(updateFailingRepo{product: existing, err: boom}, fakeS3Client{}, fakeCategoryProvider{}, time.Hour, 5*1024*1024, m, moderation.NewNoopProvider())

		_, err = svc.UpdateProduct(t.Context(), validUpdateProductRequest("p1", 1))
		require.Error(t, err)
		assert.ErrorIs(t, err, boom)
	})

	t.Run("Missing version is rejected", func(t *testing.T) {
		svc, repo := newTestService(t)
		seedUpdateProduct(t, repo)

		req := validUpdateProductRequest("p1", 1)
		req.Version = 0
		_, err := svc.UpdateProduct(t.Context(), req)
		require.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrInvalidVersion)
	})

	t.Run("Stale version is rejected", func(t *testing.T) {
		svc, repo := newTestService(t)
		seedUpdateProduct(t, repo)

		// another writer updates first
		_, err := svc.UpdateProduct(t.Context(), validUpdateProductRequest("p1", 1))
		require.NoError(t, err)

		// stale copy uses old version 1
		req := validUpdateProductRequest("p1", 1)
		req.Name = "Gadget"
		_, err = svc.UpdateProduct(t.Context(), req)
		require.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrProductOptimisticLock)
	})

	t.Run("Stale ETag is rejected", func(t *testing.T) {
		svc, repo := newTestService(t)
		seedUpdateProduct(t, repo)

		req := validUpdateProductRequest("p1", 99)
		_, err := svc.UpdateProduct(t.Context(), req)
		require.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrProductOptimisticLock)
	})

	t.Run("Empty SKU is rejected", func(t *testing.T) {
		svc, repo := newTestService(t)
		seedUpdateProduct(t, repo)

		req := validUpdateProductRequest("p1", 1)
		req.SKU = "   "
		_, err := svc.UpdateProduct(t.Context(), req)
		require.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrEmptySKU)
	})

	t.Run("Product got updated with no problems", func(t *testing.T) {
		svc, repo := newTestService(t)
		seedUpdateProduct(t, repo)

		created, _ := repo.GetByID(t.Context(), "p1")
		createdAt := created.CreatedAt

		got, err := svc.UpdateProduct(t.Context(), validUpdateProductRequest("p1", 1))
		require.NoError(t, err)
		assert.Equal(t, "p1", got.ID)
		assert.Equal(t, "Gadget", got.Name)
		assert.Equal(t, int64(750), got.PriceCents)
		assert.Equal(t, 3, got.Stock)
		assert.Equal(t, 2, got.Version)
		assert.Equal(t, createdAt, got.CreatedAt)

		stored, err := repo.GetByID(t.Context(), "p1")
		require.NoError(t, err)
		assert.Equal(t, got, stored)
	})

	t.Run("Product is updated using the ETag from GET", func(t *testing.T) {
		svc, repo := newTestService(t)
		seedUpdateProduct(t, repo)

		got, err := svc.UpdateProduct(t.Context(), validUpdateProductRequest("p1", 1))
		require.NoError(t, err)
		assert.Equal(t, 2, got.Version)
	})

	t.Run("Product name is trimmed on update", func(t *testing.T) {
		svc, repo := newTestService(t)
		seedUpdateProduct(t, repo)

		req := validUpdateProductRequest("p1", 1)
		req.Name = "  Gadget  "
		got, err := svc.UpdateProduct(t.Context(), req)
		require.NoError(t, err)
		assert.Equal(t, "Gadget", got.Name)
	})

	t.Run("Product got updated with description", func(t *testing.T) {
		svc, repo := newTestService(t)
		seedUpdateProduct(t, repo)

		req := validUpdateProductRequest("p1", 1)
		req.Description = "Updated description"
		got, err := svc.UpdateProduct(t.Context(), req)
		require.NoError(t, err)
		assert.Equal(t, "Updated description", got.Description)
	})

	t.Run("Product got updated with SKU", func(t *testing.T) {
		svc, repo := newTestService(t)
		seedUpdateProduct(t, repo)

		req := validUpdateProductRequest("p1", 1)
		req.SKU = "NEW-SKU"
		got, err := svc.UpdateProduct(t.Context(), req)
		require.NoError(t, err)
		assert.Equal(t, "NEW-SKU", got.SKU)
	})

	t.Run("Product got updated with details", func(t *testing.T) {
		svc, repo := newTestService(t)
		seedUpdateProduct(t, repo)

		req := validUpdateProductRequest("p1", 1)
		req.Details = []domain.ProductDetail{{Name: "Color", Value: "Blue"}}
		got, err := svc.UpdateProduct(t.Context(), req)
		require.NoError(t, err)
		assert.Equal(t, []domain.ProductDetail{{Name: "Color", Value: "Blue"}}, got.Details)
	})
}
