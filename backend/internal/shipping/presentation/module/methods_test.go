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

func TestListMethods(t *testing.T) {
	m1, _ := domain.NewShippingMethod("Standard", "Standard delivery", true, 3, 5)
	m2, _ := domain.NewShippingMethod("Express", "Express delivery", true, 1, 2)

	t.Run("returns all methods mapped to client type", func(t *testing.T) {
		repo := memory.NewRepository()
		require.NoError(t, repo.CreateMethod(t.Context(), m1))
		require.NoError(t, repo.CreateMethod(t.Context(), m2))
		adapter := New(application.NewService(repo, testModules{}, nil, metrics.Noop{}, time.Hour))

		got, err := adapter.ListMethods(t.Context())
		require.NoError(t, err)
		require.Len(t, got, 2)
		assert.Equal(t, m2.ID, got[0].ID)
		assert.Equal(t, "Express", got[0].Name)
		assert.Equal(t, m1.ID, got[1].ID)
		assert.Equal(t, "Standard", got[1].Name)
	})

	t.Run("returns empty slice when no methods exist", func(t *testing.T) {
		adapter := New(application.NewService(memory.NewRepository(), testModules{}, nil, metrics.Noop{}, time.Hour))

		got, err := adapter.ListMethods(t.Context())
		require.NoError(t, err)
		assert.Empty(t, got)
	})
}

func TestCreateMethod(t *testing.T) {
	t.Run("creates a method and returns client type", func(t *testing.T) {
		repo := memory.NewRepository()
		adapter := New(application.NewService(repo, testModules{}, nil, metrics.Noop{}, time.Hour))

		req := shippingclient.CreateMethodRequest{
			Name:             "Overnight",
			Description:      "Overnight delivery",
			Active:           true,
			EstimatedMinDays: 1,
			EstimatedMaxDays: 1,
		}
		got, err := adapter.CreateMethod(t.Context(), req)
		require.NoError(t, err)
		assert.Equal(t, "Overnight", got.Name)
		assert.Equal(t, "Overnight delivery", got.Description)
		assert.True(t, got.Active)
		assert.Equal(t, 1, got.EstimatedMinDays)
		assert.Equal(t, 1, got.EstimatedMaxDays)
		assert.NotEmpty(t, got.ID)
		assert.NotEmpty(t, got.CreatedAt)
		assert.NotEmpty(t, got.UpdatedAt)
	})
}

func TestUpdateMethod(t *testing.T) {
	m, _ := domain.NewShippingMethod("Standard", "Standard delivery", true, 3, 5)

	t.Run("updates a method and returns client type", func(t *testing.T) {
		repo := memory.NewRepository()
		require.NoError(t, repo.CreateMethod(t.Context(), m))
		adapter := New(application.NewService(repo, testModules{}, nil, metrics.Noop{}, time.Hour))

		req := shippingclient.UpdateMethodRequest{
			Name:             "Standard Updated",
			Description:      "Updated description",
			Active:           false,
			EstimatedMinDays: 2,
			EstimatedMaxDays: 4,
		}
		got, err := adapter.UpdateMethod(t.Context(), m.ID, req)
		require.NoError(t, err)
		assert.Equal(t, "Standard Updated", got.Name)
		assert.Equal(t, "Updated description", got.Description)
		assert.False(t, got.Active)
		assert.Equal(t, 2, got.EstimatedMinDays)
		assert.Equal(t, 4, got.EstimatedMaxDays)
	})

	t.Run("returns not-found error for missing method", func(t *testing.T) {
		adapter := New(application.NewService(memory.NewRepository(), testModules{}, nil, metrics.Noop{}, time.Hour))

		_, err := adapter.UpdateMethod(t.Context(), "missing", shippingclient.UpdateMethodRequest{})
		assert.ErrorIs(t, err, shippingclient.ErrMethodNotFound)
	})
}
