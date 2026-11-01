//go:build integration

package test

import (
	"minimart/internal/catalog/domain"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestReserveStockIntegration(t *testing.T) {
	env := newCatalogEnv(t)

	t.Run("Reservation has no items", func(t *testing.T) {
		err := env.Svc.ReserveStock(t.Context(), "o1", nil)
		require.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrInvalidReservation)
	})

	t.Run("Reservation quantity is not positive", func(t *testing.T) {
		seedCatalogProduct(t, env, "p1", 5)
		err := env.Svc.ReserveStock(t.Context(), "o1", []domain.ReservationItem{{ProductID: "p1", Quantity: 0}})
		require.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrInvalidReservation)
	})

	t.Run("Product does not exist", func(t *testing.T) {
		err := env.Svc.ReserveStock(t.Context(), "o1", []domain.ReservationItem{{ProductID: "nope", Quantity: 1}})
		require.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrProductNotFound)
	})

	t.Run("Product does not have enough stock", func(t *testing.T) {
		seedCatalogProduct(t, env, "p1", 5)
		err := env.Svc.ReserveStock(t.Context(), "o1", []domain.ReservationItem{{ProductID: "p1", Quantity: 6}})
		require.Error(t, err)
		require.ErrorIs(t, err, domain.ErrInsufficientStock)

		p, err := env.Repo.GetByID(t.Context(), "p1")
		require.NoError(t, err)
		assert.Equal(t, 5, p.Stock)
	})

	t.Run("The reservation is all-or-nothing", func(t *testing.T) {
		seedCatalogProduct(t, env, "p1", 5)
		seedCatalogProduct(t, env, "p2", 1)
		err := env.Svc.ReserveStock(t.Context(), "o1", []domain.ReservationItem{
			{ProductID: "p1", Quantity: 2},
			{ProductID: "p2", Quantity: 2},
		})
		require.Error(t, err)
		require.ErrorIs(t, err, domain.ErrInsufficientStock)

		p1, err := env.Repo.GetByID(t.Context(), "p1")
		require.NoError(t, err)
		assert.Equal(t, 5, p1.Stock)
	})

	t.Run("Re-reserving the same order does not reserve twice", func(t *testing.T) {
		seedCatalogProduct(t, env, "p1", 5)
		items := []domain.ReservationItem{{ProductID: "p1", Quantity: 2}}
		require.NoError(t, env.Svc.ReserveStock(t.Context(), "o1", items))
		require.NoError(t, env.Svc.ReserveStock(t.Context(), "o1", items))

		p, err := env.Repo.GetByID(t.Context(), "p1")
		require.NoError(t, err)
		assert.Equal(t, 3, p.Stock)
	})

	t.Run("Reserve decrements the product's stock", func(t *testing.T) {
		seedCatalogProduct(t, env, "p1", 5)
		err := env.Svc.ReserveStock(t.Context(), "o1", []domain.ReservationItem{{ProductID: "p1", Quantity: 2}})
		require.NoError(t, err)

		p, err := env.Repo.GetByID(t.Context(), "p1")
		require.NoError(t, err)
		assert.Equal(t, 3, p.Stock)
	})
}
