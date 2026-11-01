package application

import (
	"context"
	"errors"
	"minimart/internal/catalog/domain"
	"minimart/pkg/metrics"
	"minimart/pkg/moderation"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type failingRepo struct {
	catalogRepoStub
	err error
}

func (f failingRepo) Save(context.Context, domain.Product) error { return f.err }

func TestCreateProduct(t *testing.T) {
	t.Run("Blank name is rejected", func(t *testing.T) {
		svc, _ := newTestService(t)

		req := validCreateProductRequest()
		req.Name = "   "
		_, err := svc.CreateProduct(t.Context(), req)
		require.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrEmptyName)
	})

	t.Run("Non-positive price is rejected", func(t *testing.T) {
		svc, _ := newTestService(t)

		req := validCreateProductRequest()
		req.PriceCents = 0
		_, err := svc.CreateProduct(t.Context(), req)
		require.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrNonPositivePrice)
	})

	t.Run("Negative stock is rejected", func(t *testing.T) {
		svc, _ := newTestService(t)

		req := validCreateProductRequest()
		req.Stock = -1
		_, err := svc.CreateProduct(t.Context(), req)
		require.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrNegativeStock)
	})

	t.Run("Storage failure", func(t *testing.T) {
		boom := errors.New("disk on fire")
		svc := NewService(failingRepo{err: boom}, fakeS3Client{}, fakeCategoryProvider{}, time.Hour, 5*1024*1024, metrics.Noop{}, moderation.NewNoopProvider())

		_, err := svc.CreateProduct(t.Context(), validCreateProductRequest())
		require.Error(t, err)
		assert.ErrorIs(t, err, boom)
	})

	t.Run("Empty SKU is rejected", func(t *testing.T) {
		svc, _ := newTestService(t)

		req := validCreateProductRequest()
		req.SKU = "   "
		_, err := svc.CreateProduct(t.Context(), req)
		require.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrEmptySKU)
	})

	t.Run("Duplicate SKU is rejected", func(t *testing.T) {
		svc, repo := newTestService(t)

		// seed product with SKU SKU-123
		p, err := domain.NewProduct("seed", "Existing", 500, 3, "cat-1", "SKU-123", "", nil)
		require.NoError(t, err)
		require.NoError(t, repo.Save(t.Context(), p))

		req := validCreateProductRequest()
		req.SKU = "SKU-123"
		_, err = svc.CreateProduct(t.Context(), req)
		require.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrDuplicateSKU)
	})

	t.Run("Product got created with no problems", func(t *testing.T) {
		svc, repo := newTestService(t)

		created, err := svc.CreateProduct(t.Context(), validCreateProductRequest())
		require.NoError(t, err)

		stored, err := repo.GetByID(t.Context(), created.ID)
		require.NoError(t, err)
		assert.NotEmpty(t, stored.ID)
		assert.Equal(t, "Widget", stored.Name)
		assert.Equal(t, int64(500), stored.PriceCents)
		assert.Equal(t, 3, stored.Stock)
		assert.False(t, stored.CreatedAt.IsZero())
	})

	t.Run("Product name is trimmed on creation", func(t *testing.T) {
		svc, repo := newTestService(t)

		req := validCreateProductRequest()
		req.Name = "  Widget  "
		created, err := svc.CreateProduct(t.Context(), req)
		require.NoError(t, err)

		stored, err := repo.GetByID(t.Context(), created.ID)
		require.NoError(t, err)
		assert.Equal(t, "Widget", stored.Name)
	})

	t.Run("Product got created with description", func(t *testing.T) {
		svc, repo := newTestService(t)

		req := validCreateProductRequest()
		req.Description = "A great widget"
		created, err := svc.CreateProduct(t.Context(), req)
		require.NoError(t, err)

		stored, err := repo.GetByID(t.Context(), created.ID)
		require.NoError(t, err)
		assert.Equal(t, "A great widget", stored.Description)
	})

	t.Run("Product got created with SKU", func(t *testing.T) {
		svc, repo := newTestService(t)

		req := validCreateProductRequest()
		req.SKU = "SKU-123"
		created, err := svc.CreateProduct(t.Context(), req)
		require.NoError(t, err)

		stored, err := repo.GetByID(t.Context(), created.ID)
		require.NoError(t, err)
		assert.Equal(t, "SKU-123", stored.SKU)
	})

	t.Run("Product got created with details", func(t *testing.T) {
		svc, repo := newTestService(t)

		req := validCreateProductRequest()
		req.Details = []domain.ProductDetail{{Name: "Color", Value: "Red"}, {Name: "Size", Value: "Large"}}
		created, err := svc.CreateProduct(t.Context(), req)
		require.NoError(t, err)

		stored, err := repo.GetByID(t.Context(), created.ID)
		require.NoError(t, err)
		assert.Equal(t, []domain.ProductDetail{{Name: "Color", Value: "Red"}, {Name: "Size", Value: "Large"}}, stored.Details)
	})
}
