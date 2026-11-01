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

func TestReleaseStock(t *testing.T) {
	t.Run("restores the reserved quantity", func(t *testing.T) {
		repo := memory.New()
		seedProduct(t, repo, "p1", 5)
		svc := NewService(repo, fakeS3Client{}, fakeCategoryProvider{}, time.Hour, 5*1024*1024, metrics.Noop{}, moderation.NewNoopProvider())

		require.NoError(t, svc.ReserveStock(t.Context(), "o1", []domain.ReservationItem{{ProductID: "p1", Quantity: 2}}))
		require.NoError(t, svc.ReleaseStock(t.Context(), "o1"))

		p, err := repo.GetByID(t.Context(), "p1")
		require.NoError(t, err)
		assert.Equal(t, 5, p.Stock, "released stock returns to the product")
	})

	t.Run("releasing twice is idempotent", func(t *testing.T) {
		repo := memory.New()
		seedProduct(t, repo, "p1", 5)
		svc := NewService(repo, fakeS3Client{}, fakeCategoryProvider{}, time.Hour, 5*1024*1024, metrics.Noop{}, moderation.NewNoopProvider())

		require.NoError(t, svc.ReserveStock(t.Context(), "o1", []domain.ReservationItem{{ProductID: "p1", Quantity: 2}}))
		require.NoError(t, svc.ReleaseStock(t.Context(), "o1"))
		require.NoError(t, svc.ReleaseStock(t.Context(), "o1"))

		p, err := repo.GetByID(t.Context(), "p1")
		require.NoError(t, err)
		assert.Equal(t, 5, p.Stock, "a redelivered release does not add stock twice")
	})
}

func TestReleaseStockRecordsMetrics(t *testing.T) {
	repo := memory.New()
	seedProduct(t, repo, "p1", 5)
	m := &countingMetrics{}
	svc := NewService(repo, fakeS3Client{}, fakeCategoryProvider{}, time.Hour, 5*1024*1024, m, moderation.NewNoopProvider())

	require.NoError(t, svc.ReserveStock(t.Context(), "o1", []domain.ReservationItem{{ProductID: "p1", Quantity: 1}}))
	require.NoError(t, svc.ReleaseStock(t.Context(), "o1"))
	assert.Equal(t, []string{metrics.StatusSuccess}, m.released)
}
