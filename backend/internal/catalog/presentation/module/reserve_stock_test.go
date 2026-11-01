package module

import (
	"minimart/internal/catalog/application"
	"minimart/internal/catalog/domain"
	"minimart/internal/catalog/infrastructure/memory"
	"minimart/internal/catalog/presentation/module/client"
	"minimart/internal/catalog/testkit"
	"minimart/pkg/metrics"
	"minimart/pkg/moderation"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestReserveStock(t *testing.T) {
	repo := memory.New()
	widget, err := domain.NewProduct("p1", "Widget", 500, 1, "cat-1", "", "", nil)
	require.NoError(t, err)
	require.NoError(t, repo.Save(t.Context(), widget))
	adapter := New(application.NewService(repo, testkit.NewFakeS3Client(), fakeCategoryProvider{}, time.Hour, 5*1024*1024, metrics.Noop{}, moderation.NewNoopProvider()))

	t.Run("reserves available stock", func(t *testing.T) {
		require.NoError(t, adapter.ReserveStock(t.Context(), "o1", []client.LineItem{{ProductID: "p1", Quantity: 1}}))
	})

	t.Run("translates insufficient stock into the client sentinel", func(t *testing.T) {
		err := adapter.ReserveStock(t.Context(), "o2", []client.LineItem{{ProductID: "p1", Quantity: 1}})
		require.ErrorIs(t, err, client.ErrInsufficientStock)
	})

	t.Run("translates an unknown product into the client sentinel", func(t *testing.T) {
		err := adapter.ReserveStock(t.Context(), "o3", []client.LineItem{{ProductID: "nope", Quantity: 1}})
		require.ErrorIs(t, err, client.ErrProductNotFound)
	})
}
