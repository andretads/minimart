//go:build integration

package test

import (
	"minimart/internal/catalog/application"
	"minimart/internal/catalog/domain"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCreateProductIntegration(t *testing.T) {
	env := newCatalogEnv(t)

	t.Run("Blank name is rejected", func(t *testing.T) {
		req := validCreateProductReq()
		req.Name = "   "
		_, err := env.Svc.CreateProduct(t.Context(), req)
		require.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrEmptyName)
	})

	t.Run("Non-positive price is rejected", func(t *testing.T) {
		req := validCreateProductReq()
		req.PriceCents = 0
		_, err := env.Svc.CreateProduct(t.Context(), req)
		require.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrNonPositivePrice)
	})

	t.Run("Negative stock is rejected", func(t *testing.T) {
		req := validCreateProductReq()
		req.Stock = -1
		_, err := env.Svc.CreateProduct(t.Context(), req)
		require.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrNegativeStock)
	})

	t.Run("Storage failure", func(t *testing.T) {
		t.Skip("storage failure not easily simulated with real DB")
	})

	t.Run("Empty SKU is rejected", func(t *testing.T) {
		req := validCreateProductReq()
		req.SKU = "   "
		_, err := env.Svc.CreateProduct(t.Context(), req)
		require.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrEmptySKU)
	})

	t.Run("Duplicate SKU is rejected", func(t *testing.T) {
		req := validCreateProductReq()
		req.SKU = "SKU-IT-DUP"
		_, err := env.Svc.CreateProduct(t.Context(), req)
		require.NoError(t, err)

		req2 := validCreateProductReq()
		req2.SKU = "SKU-IT-DUP"
		_, err = env.Svc.CreateProduct(t.Context(), req2)
		require.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrDuplicateSKU)
	})

	t.Run("Product got created with no problems", func(t *testing.T) {
		req := validCreateProductReq()
		created, err := env.Svc.CreateProduct(t.Context(), req)
		require.NoError(t, err)
		assert.NotEmpty(t, created.ID)
		assert.Equal(t, "Widget", created.Name)
		assert.Equal(t, int64(500), created.PriceCents)
		assert.Equal(t, 3, created.Stock)
		assert.False(t, created.CreatedAt.IsZero())
	})

	t.Run("Product name is trimmed on creation", func(t *testing.T) {
		req := validCreateProductReq()
		req.Name = "  Widget  "
		created, err := env.Svc.CreateProduct(t.Context(), req)
		require.NoError(t, err)
		assert.Equal(t, "Widget", created.Name)
	})

	t.Run("Product got created with description", func(t *testing.T) {
		req := validCreateProductReq()
		req.Description = "A great widget"
		created, err := env.Svc.CreateProduct(t.Context(), req)
		require.NoError(t, err)
		assert.Equal(t, "A great widget", created.Description)
	})

	t.Run("Product got created with SKU", func(t *testing.T) {
		req := validCreateProductReq()
		req.SKU = "SKU-123"
		created, err := env.Svc.CreateProduct(t.Context(), req)
		require.NoError(t, err)
		assert.Equal(t, "SKU-123", created.SKU)
	})

	t.Run("Product got created with details", func(t *testing.T) {
		req := validCreateProductReq()
		req.Details = []domain.ProductDetail{{Name: "Color", Value: "Red"}, {Name: "Size", Value: "Large"}}
		created, err := env.Svc.CreateProduct(t.Context(), req)
		require.NoError(t, err)
		assert.Equal(t, []domain.ProductDetail{{Name: "Color", Value: "Red"}, {Name: "Size", Value: "Large"}}, created.Details)
	})
}

func validCreateProductReq() application.CreateProductRequest {
	return application.CreateProductRequest{Name: "Widget", PriceCents: 500, Stock: 3, CategoryID: "cat-1"}
}
