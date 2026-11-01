//go:build integration

package test

import (
	"minimart/internal/catalog/domain"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestReleaseStockIntegration(t *testing.T) {
	env := newCatalogEnv(t)

	t.Run("Releasing twice is idempotent", func(t *testing.T) {
		seedCatalogProduct(t, env, "p1", 5)
		require.NoError(t, env.Svc.ReserveStock(t.Context(), "o1", []domain.ReservationItem{{ProductID: "p1", Quantity: 2}}))
		require.NoError(t, env.Svc.ReleaseStock(t.Context(), "o1"))
		require.NoError(t, env.Svc.ReleaseStock(t.Context(), "o1"))

		p, err := env.Repo.GetByID(t.Context(), "p1")
		require.NoError(t, err)
		assert.Equal(t, 5, p.Stock)
	})

	t.Run("Release restores the product's stock", func(t *testing.T) {
		seedCatalogProduct(t, env, "p1", 5)
		require.NoError(t, env.Svc.ReserveStock(t.Context(), "o1", []domain.ReservationItem{{ProductID: "p1", Quantity: 2}}))
		require.NoError(t, env.Svc.ReleaseStock(t.Context(), "o1"))

		p, err := env.Repo.GetByID(t.Context(), "p1")
		require.NoError(t, err)
		assert.Equal(t, 5, p.Stock)
	})
	_ = domain.ErrProductNotFound
}
