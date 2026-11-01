package module

import (
	"errors"
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

func TestGetMethod(t *testing.T) {
	m, _ := domain.NewShippingMethod("Standard", "Standard delivery", true, 3, 5)

	t.Run("maps a domain method into the client type", func(t *testing.T) {
		repo := memory.NewRepository()
		require.NoError(t, repo.CreateMethod(t.Context(), m))
		adapter := New(application.NewService(repo, testModules{}, nil, metrics.Noop{}, time.Hour))

		got, err := adapter.GetMethod(t.Context(), m.ID)
		require.NoError(t, err)
		assert.Equal(t, m.ID, got.ID)
		assert.Equal(t, "Standard", got.Name)
		assert.Equal(t, "Standard delivery", got.Description)
		assert.True(t, got.Active)
		assert.Equal(t, 3, got.EstimatedMinDays)
		assert.Equal(t, 5, got.EstimatedMaxDays)
		assert.NotEmpty(t, got.CreatedAt)
		assert.NotEmpty(t, got.UpdatedAt)
	})

	t.Run("translates not-found into the client sentinel", func(t *testing.T) {
		adapter := New(application.NewService(memory.NewRepository(), testModules{}, nil, metrics.Noop{}, time.Hour))

		_, err := adapter.GetMethod(t.Context(), "missing")
		assert.ErrorIs(t, err, shippingclient.ErrMethodNotFound)
	})

	t.Run("passes through unexpected errors unchanged", func(t *testing.T) {
		boom := errors.New("boom")
		adapter := New(application.NewService(failingRepo{err: boom}, testModules{}, nil, metrics.Noop{}, time.Hour))

		_, err := adapter.GetMethod(t.Context(), "m1")
		require.ErrorIs(t, err, boom)
		assert.NotErrorIs(t, err, shippingclient.ErrMethodNotFound)
	})
}
