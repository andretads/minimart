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
	t.Run("Releasing twice is idempotent", func(t *testing.T) {
		repo := memory.New()
		seedProduct(t, repo, "p1", 5)
		svc := NewService(repo, fakeS3Client{}, fakeCategoryProvider{}, time.Hour, 5*1024*1024, metrics.Noop{}, moderation.NewNoopProvider())

		require.NoError(t, svc.ReserveStock(t.Context(), "o1", []domain.ReservationItem{{ProductID: "p1", Quantity: 2}}))
		require.NoError(t, svc.ReleaseStock(t.Context(), "o1"))
		require.NoError(t, svc.ReleaseStock(t.Context(), "o1"))

		p, err := repo.GetByID(t.Context(), "p1")
		require.NoError(t, err)
		assert.Equal(t, 5, p.Stock)
	})

	t.Run("Release restores the product's stock", func(t *testing.T) {
		repo := memory.New()
		seedProduct(t, repo, "p1", 5)
		svc := NewService(repo, fakeS3Client{}, fakeCategoryProvider{}, time.Hour, 5*1024*1024, metrics.Noop{}, moderation.NewNoopProvider())

		require.NoError(t, svc.ReserveStock(t.Context(), "o1", []domain.ReservationItem{{ProductID: "p1", Quantity: 2}}))
		require.NoError(t, svc.ReleaseStock(t.Context(), "o1"))

		p, err := repo.GetByID(t.Context(), "p1")
		require.NoError(t, err)
		assert.Equal(t, 5, p.Stock)
	})
}
