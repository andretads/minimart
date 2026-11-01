package application

import (
	"minimart/internal/catalog/domain"
	"minimart/internal/catalog/infrastructure/memory"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func seedProduct(t *testing.T, repo *memory.Repository, id string, stock int) {
	t.Helper()
	p, err := domain.NewProduct(id, "Widget", 500, stock, "cat-1", "", "", nil)
	require.NoError(t, err)
	require.NoError(t, repo.Save(t.Context(), p))
}

func TestReserveStock(t *testing.T) {
	t.Run("Reservation has no items", func(t *testing.T) {
		svc, _ := newTestService(t)

		err := svc.ReserveStock(t.Context(), "o1", nil)
		require.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrInvalidReservation)
	})

	t.Run("Reservation quantity is not positive", func(t *testing.T) {
		svc, repo := newTestService(t)
		seedProduct(t, repo, "p1", 5)

		err := svc.ReserveStock(t.Context(), "o1", []domain.ReservationItem{{ProductID: "p1", Quantity: 0}})
		require.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrInvalidReservation)
	})

	t.Run("Product does not exist", func(t *testing.T) {
		svc, _ := newTestService(t)

		err := svc.ReserveStock(t.Context(), "o1", []domain.ReservationItem{{ProductID: "nope", Quantity: 1}})
		require.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrProductNotFound)
	})

	t.Run("Product does not have enough stock", func(t *testing.T) {
		svc, repo := newTestService(t)
		seedProduct(t, repo, "p1", 5)

		err := svc.ReserveStock(t.Context(), "o1", []domain.ReservationItem{{ProductID: "p1", Quantity: 6}})
		require.Error(t, err)
		require.ErrorIs(t, err, domain.ErrInsufficientStock)

		p, err := repo.GetByID(t.Context(), "p1")
		require.NoError(t, err)
		assert.Equal(t, 5, p.Stock)
	})

	t.Run("The reservation is all-or-nothing", func(t *testing.T) {
		svc, repo := newTestService(t)
		seedProduct(t, repo, "p1", 5)
		seedProduct(t, repo, "p2", 1)

		err := svc.ReserveStock(t.Context(), "o1", []domain.ReservationItem{
			{ProductID: "p1", Quantity: 2},
			{ProductID: "p2", Quantity: 2},
		})
		require.Error(t, err)
		require.ErrorIs(t, err, domain.ErrInsufficientStock)

		p1, err := repo.GetByID(t.Context(), "p1")
		require.NoError(t, err)
		assert.Equal(t, 5, p1.Stock)
	})

	t.Run("Re-reserving the same order does not reserve twice", func(t *testing.T) {
		svc, repo := newTestService(t)
		seedProduct(t, repo, "p1", 5)

		items := []domain.ReservationItem{{ProductID: "p1", Quantity: 2}}
		require.NoError(t, svc.ReserveStock(t.Context(), "o1", items))
		require.NoError(t, svc.ReserveStock(t.Context(), "o1", items))

		p, err := repo.GetByID(t.Context(), "p1")
		require.NoError(t, err)
		assert.Equal(t, 3, p.Stock)
	})

	t.Run("Reserve decrements the product's stock", func(t *testing.T) {
		svc, repo := newTestService(t)
		seedProduct(t, repo, "p1", 5)

		err := svc.ReserveStock(t.Context(), "o1", []domain.ReservationItem{{ProductID: "p1", Quantity: 2}})
		require.NoError(t, err)

		p, err := repo.GetByID(t.Context(), "p1")
		require.NoError(t, err)
		assert.Equal(t, 3, p.Stock)
	})
}
