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

func TestListZones(t *testing.T) {
	z1, _ := domain.NewShippingZone("Brazil", "BR", "", "", "", "")
	z2, _ := domain.NewShippingZone("USA", "US", "", "", "", "")

	t.Run("returns all zones mapped to client type", func(t *testing.T) {
		repo := memory.NewRepository()
		require.NoError(t, repo.CreateZone(t.Context(), z1))
		require.NoError(t, repo.CreateZone(t.Context(), z2))
		adapter := New(application.NewService(repo, testModules{}, nil, metrics.Noop{}, time.Hour))

		got, err := adapter.ListZones(t.Context())
		require.NoError(t, err)
		require.Len(t, got, 2)

		// Check both zones are present without assuming order
		names := make(map[string]bool)
		countries := make(map[string]bool)
		for _, z := range got {
			names[z.Name] = true
			countries[z.Country] = true
		}
		assert.True(t, names["Brazil"])
		assert.True(t, names["USA"])
		assert.True(t, countries["BR"])
		assert.True(t, countries["US"])
	})

	t.Run("returns empty slice when no zones exist", func(t *testing.T) {
		adapter := New(application.NewService(memory.NewRepository(), testModules{}, nil, metrics.Noop{}, time.Hour))

		got, err := adapter.ListZones(t.Context())
		require.NoError(t, err)
		assert.Empty(t, got)
	})
}

func TestGetZone(t *testing.T) {
	z, _ := domain.NewShippingZone("Brazil", "BR", "SP", "Sao Paulo", "01000", "01999")

	t.Run("returns zone mapped to client type", func(t *testing.T) {
		repo := memory.NewRepository()
		require.NoError(t, repo.CreateZone(t.Context(), z))
		adapter := New(application.NewService(repo, testModules{}, nil, metrics.Noop{}, time.Hour))

		got, err := adapter.GetZone(t.Context(), z.ID)
		require.NoError(t, err)
		assert.Equal(t, z.ID, got.ID)
		assert.Equal(t, "Brazil", got.Name)
		assert.Equal(t, "BR", got.Country)
		assert.Equal(t, "SP", got.State)
		assert.Equal(t, "Sao Paulo", got.City)
		assert.Equal(t, "01000", got.PostalCodeStart)
		assert.Equal(t, "01999", got.PostalCodeEnd)
	})

	t.Run("returns not-found error for missing zone", func(t *testing.T) {
		adapter := New(application.NewService(memory.NewRepository(), testModules{}, nil, metrics.Noop{}, time.Hour))

		_, err := adapter.GetZone(t.Context(), "missing")
		assert.ErrorIs(t, err, shippingclient.ErrZoneNotFound)
	})
}

func TestCreateZone(t *testing.T) {
	t.Run("creates a zone and returns client type", func(t *testing.T) {
		repo := memory.NewRepository()
		adapter := New(application.NewService(repo, testModules{}, nil, metrics.Noop{}, time.Hour))

		req := shippingclient.CreateZoneRequest{
			Name:            "Canada",
			Country:         "CA",
			State:           "ON",
			City:            "Toronto",
			PostalCodeStart: "M1B",
			PostalCodeEnd:   "M9Z",
		}
		got, err := adapter.CreateZone(t.Context(), req)
		require.NoError(t, err)
		assert.Equal(t, "Canada", got.Name)
		assert.Equal(t, "CA", got.Country)
		assert.Equal(t, "ON", got.State)
		assert.Equal(t, "Toronto", got.City)
		assert.Equal(t, "M1B", got.PostalCodeStart)
		assert.Equal(t, "M9Z", got.PostalCodeEnd)
		assert.NotEmpty(t, got.ID)
		assert.NotEmpty(t, got.CreatedAt)
		assert.NotEmpty(t, got.UpdatedAt)
	})
}

func TestUpdateZone(t *testing.T) {
	z, _ := domain.NewShippingZone("Brazil", "BR", "SP", "Sao Paulo", "01000", "01999")

	t.Run("updates a zone and returns client type", func(t *testing.T) {
		repo := memory.NewRepository()
		require.NoError(t, repo.CreateZone(t.Context(), z))
		adapter := New(application.NewService(repo, testModules{}, nil, metrics.Noop{}, time.Hour))

		req := shippingclient.UpdateZoneRequest{
			Name:            "Brazil Updated",
			Country:         "BR",
			State:           "RJ",
			City:            "Rio de Janeiro",
			PostalCodeStart: "20000",
			PostalCodeEnd:   "20999",
		}
		got, err := adapter.UpdateZone(t.Context(), z.ID, req)
		require.NoError(t, err)
		assert.Equal(t, "Brazil Updated", got.Name)
		assert.Equal(t, "RJ", got.State)
		assert.Equal(t, "Rio de Janeiro", got.City)
		assert.Equal(t, "20000", got.PostalCodeStart)
		assert.Equal(t, "20999", got.PostalCodeEnd)
	})

	t.Run("returns not-found error for missing zone", func(t *testing.T) {
		adapter := New(application.NewService(memory.NewRepository(), testModules{}, nil, metrics.Noop{}, time.Hour))

		_, err := adapter.UpdateZone(t.Context(), "missing", shippingclient.UpdateZoneRequest{})
		assert.ErrorIs(t, err, shippingclient.ErrZoneNotFound)
	})
}
