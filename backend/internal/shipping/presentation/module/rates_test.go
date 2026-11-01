package module

import (
	"minimart/internal/shipping/application"
	"minimart/internal/shipping/domain"
	"minimart/internal/shipping/infrastructure/memory"
	shippingclient "minimart/internal/shipping/presentation/module/client"
	"minimart/pkg/metrics"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestListRates(t *testing.T) {
	zone, _ := domain.NewShippingZone("Brazil", "BR", "", "", "", "")
	method, _ := domain.NewShippingMethod("Standard", "Standard delivery", true, 3, 5)
	require.NoError(t, memory.NewRepository().CreateZone(t.Context(), zone))
	require.NoError(t, memory.NewRepository().CreateMethod(t.Context(), method))

	r1, _ := domain.NewShippingRate(zone.ID, method.ID, 599, 5000, true)
	r2, _ := domain.NewShippingRate(zone.ID, method.ID, 999, 10000, true)

	t.Run("returns all rates for zone and method mapped to client type", func(t *testing.T) {
		repo := memory.NewRepository()
		require.NoError(t, repo.CreateZone(t.Context(), zone))
		require.NoError(t, repo.CreateMethod(t.Context(), method))
		require.NoError(t, repo.CreateRate(t.Context(), r1))
		require.NoError(t, repo.CreateRate(t.Context(), r2))
		adapter := New(application.NewService(repo, testModules{}, nil, metrics.Noop{}, time.Hour))

		got, err := adapter.ListRates(t.Context(), zone.ID, method.ID)
		require.NoError(t, err)
		require.Len(t, got, 2)
		assert.Equal(t, r1.ID, got[0].ID)
		assert.Equal(t, int64(599), got[0].BaseCostCents)
		assert.Equal(t, int64(5000), got[0].FreeThresholdCents)
		assert.Equal(t, r2.ID, got[1].ID)
		assert.Equal(t, int64(999), got[1].BaseCostCents)
	})

	t.Run("returns empty slice when no rates exist", func(t *testing.T) {
		repo := memory.NewRepository()
		require.NoError(t, repo.CreateZone(t.Context(), zone))
		require.NoError(t, repo.CreateMethod(t.Context(), method))
		adapter := New(application.NewService(repo, testModules{}, nil, metrics.Noop{}, time.Hour))

		got, err := adapter.ListRates(t.Context(), zone.ID, method.ID)
		require.NoError(t, err)
		assert.Empty(t, got)
	})
}

func TestGetRate(t *testing.T) {
	zone, _ := domain.NewShippingZone("Brazil", "BR", "", "", "", "")
	method, _ := domain.NewShippingMethod("Standard", "Standard delivery", true, 3, 5)
	r, _ := domain.NewShippingRate(zone.ID, method.ID, 599, 5000, true)

	t.Run("returns rate mapped to client type", func(t *testing.T) {
		repo := memory.NewRepository()
		require.NoError(t, repo.CreateZone(t.Context(), zone))
		require.NoError(t, repo.CreateMethod(t.Context(), method))
		require.NoError(t, repo.CreateRate(t.Context(), r))
		adapter := New(application.NewService(repo, testModules{}, nil, metrics.Noop{}, time.Hour))

		got, err := adapter.GetRate(t.Context(), r.ID)
		require.NoError(t, err)
		assert.Equal(t, r.ID, got.ID)
		assert.Equal(t, zone.ID, got.ZoneID)
		assert.Equal(t, method.ID, got.MethodID)
		assert.Equal(t, int64(599), got.BaseCostCents)
		assert.Equal(t, int64(5000), got.FreeThresholdCents)
		assert.True(t, got.Active)
	})

	t.Run("returns not-found error for missing rate", func(t *testing.T) {
		adapter := New(application.NewService(memory.NewRepository(), testModules{}, nil, metrics.Noop{}, time.Hour))

		_, err := adapter.GetRate(t.Context(), "missing")
		assert.ErrorIs(t, err, shippingclient.ErrRateNotFound)
	})
}

func TestCreateRate(t *testing.T) {
	zone, _ := domain.NewShippingZone("Brazil", "BR", "", "", "", "")
	method, _ := domain.NewShippingMethod("Standard", "Standard delivery", true, 3, 5)

	t.Run("creates a rate and returns client type", func(t *testing.T) {
		repo := memory.NewRepository()
		require.NoError(t, repo.CreateZone(t.Context(), zone))
		require.NoError(t, repo.CreateMethod(t.Context(), method))
		adapter := New(application.NewService(repo, testModules{}, nil, metrics.Noop{}, time.Hour))

		req := shippingclient.CreateRateRequest{
			ZoneID:             zone.ID,
			MethodID:           method.ID,
			BaseCostCents:      799,
			FreeThresholdCents: 7500,
			Active:             true,
		}
		got, err := adapter.CreateRate(t.Context(), req)
		require.NoError(t, err)
		assert.Equal(t, zone.ID, got.ZoneID)
		assert.Equal(t, method.ID, got.MethodID)
		assert.Equal(t, int64(799), got.BaseCostCents)
		assert.Equal(t, int64(7500), got.FreeThresholdCents)
		assert.True(t, got.Active)
		assert.NotEmpty(t, got.ID)
		assert.NotEmpty(t, got.CreatedAt)
		assert.NotEmpty(t, got.UpdatedAt)
	})
}

func TestUpdateRate(t *testing.T) {
	zone, _ := domain.NewShippingZone("Brazil", "BR", "", "", "", "")
	method, _ := domain.NewShippingMethod("Standard", "Standard delivery", true, 3, 5)
	r, _ := domain.NewShippingRate(zone.ID, method.ID, 599, 5000, true)

	t.Run("updates a rate and returns client type", func(t *testing.T) {
		repo := memory.NewRepository()
		require.NoError(t, repo.CreateZone(t.Context(), zone))
		require.NoError(t, repo.CreateMethod(t.Context(), method))
		require.NoError(t, repo.CreateRate(t.Context(), r))
		adapter := New(application.NewService(repo, testModules{}, nil, metrics.Noop{}, time.Hour))

		req := shippingclient.UpdateRateRequest{
			BaseCostCents:      699,
			FreeThresholdCents: 6000,
			Active:             false,
		}
		got, err := adapter.UpdateRate(t.Context(), r.ID, req)
		require.NoError(t, err)
		assert.Equal(t, int64(699), got.BaseCostCents)
		assert.Equal(t, int64(6000), got.FreeThresholdCents)
		assert.False(t, got.Active)
	})

	t.Run("returns not-found error for missing rate", func(t *testing.T) {
		adapter := New(application.NewService(memory.NewRepository(), testModules{}, nil, metrics.Noop{}, time.Hour))

		_, err := adapter.UpdateRate(t.Context(), "missing", shippingclient.UpdateRateRequest{})
		assert.ErrorIs(t, err, shippingclient.ErrRateNotFound)
	})
}
