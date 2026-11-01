package application

import (
	"minimart/internal/catalog/domain"
	"minimart/internal/catalog/infrastructure/memory"
	"minimart/pkg/metrics"
	"minimart/pkg/moderation"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// seedProduct saves a product with the given stock and returns its id.
func seedProduct(t *testing.T, repo *memory.Repository, id string, stock int) {
	t.Helper()
	p, err := domain.NewProduct(id, "Widget", 500, stock, "cat-1", "", "", nil)
	require.NoError(t, err)
	require.NoError(t, repo.Save(t.Context(), p))
}

func TestReserveStock(t *testing.T) {
	t.Run("decrements stock for the reserved quantity", func(t *testing.T) {
		repo := memory.New()
		seedProduct(t, repo, "p1", 5)
		svc := NewService(repo, fakeS3Client{}, fakeCategoryProvider{}, time.Hour, 5*1024*1024, metrics.Noop{}, moderation.NewNoopProvider())

		err := svc.ReserveStock(t.Context(), "o1", []domain.ReservationItem{{ProductID: "p1", Quantity: 2}})
		require.NoError(t, err)

		p, err := repo.GetByID(t.Context(), "p1")
		require.NoError(t, err)
		assert.Equal(t, 3, p.Stock)
	})

	t.Run("rejects the whole reservation when one item is short", func(t *testing.T) {
		repo := memory.New()
		seedProduct(t, repo, "p1", 5)
		seedProduct(t, repo, "p2", 1)
		svc := NewService(repo, fakeS3Client{}, fakeCategoryProvider{}, time.Hour, 5*1024*1024, metrics.Noop{}, moderation.NewNoopProvider())

		err := svc.ReserveStock(t.Context(), "o1", []domain.ReservationItem{
			{ProductID: "p1", Quantity: 2},
			{ProductID: "p2", Quantity: 2},
		})
		require.ErrorIs(t, err, domain.ErrInsufficientStock)

		p1, err := repo.GetByID(t.Context(), "p1")
		require.NoError(t, err)
		assert.Equal(t, 5, p1.Stock, "nothing is reserved when any item is short")
	})

	t.Run("re-reserving the same order does not reserve twice", func(t *testing.T) {
		repo := memory.New()
		seedProduct(t, repo, "p1", 5)
		svc := NewService(repo, fakeS3Client{}, fakeCategoryProvider{}, time.Hour, 5*1024*1024, metrics.Noop{}, moderation.NewNoopProvider())

		items := []domain.ReservationItem{{ProductID: "p1", Quantity: 2}}
		require.NoError(t, svc.ReserveStock(t.Context(), "o1", items))
		require.NoError(t, svc.ReserveStock(t.Context(), "o1", items))

		p, err := repo.GetByID(t.Context(), "p1")
		require.NoError(t, err)
		assert.Equal(t, 3, p.Stock)
	})
}

func TestReserveStockRecordsMetrics(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		repo := memory.New()
		seedProduct(t, repo, "p1", 5)
		m := &countingMetrics{}
		svc := NewService(repo, fakeS3Client{}, fakeCategoryProvider{}, time.Hour, 5*1024*1024, m, moderation.NewNoopProvider())

		require.NoError(t, svc.ReserveStock(t.Context(), "o1", []domain.ReservationItem{{ProductID: "p1", Quantity: 1}}))
		assert.Equal(t, []string{metrics.StatusSuccess}, m.reserved)
	})

	t.Run("estoque insuficiente", func(t *testing.T) {
		repo := memory.New()
		seedProduct(t, repo, "p1", 1)
		m := &countingMetrics{}
		svc := NewService(repo, fakeS3Client{}, fakeCategoryProvider{}, time.Hour, 5*1024*1024, m, moderation.NewNoopProvider())

		require.ErrorIs(t, svc.ReserveStock(t.Context(), "o1", []domain.ReservationItem{{ProductID: "p1", Quantity: 2}}), domain.ErrInsufficientStock)
		assert.Equal(t, []string{metrics.StatusError}, m.reserved)
	})
}
