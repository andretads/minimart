//go:build integration

package test

import (
	"minimart/internal/catalog/domain"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestListProductsIntegration(t *testing.T) {
	env := newCatalogEnv(t)
	_, _ = env.Pool.Exec(t.Context(), "DELETE FROM stock_reservations")
	_, _ = env.Pool.Exec(t.Context(), "DELETE FROM product_images")
	_, _ = env.Pool.Exec(t.Context(), "DELETE FROM product_reviews")
	_, _ = env.Pool.Exec(t.Context(), "DELETE FROM products")

	t.Run("Empty catalog returns an empty list", func(t *testing.T) {
		got, err := env.Svc.ListProducts(t.Context())
		require.NoError(t, err)
		assert.Empty(t, got)
	})

	t.Run("All stored products are returned", func(t *testing.T) {
		seedCatalogProduct(t, env, "p1", 1)
		seedCatalogProduct(t, env, "p2", 1)
		got, err := env.Svc.ListProducts(t.Context())
		require.NoError(t, err)
		assert.Len(t, got, 2)
		_ = domain.Product{}
	})
}
