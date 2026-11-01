package application

import (
	"minimart/internal/shipping/domain"
	"minimart/internal/shipping/infrastructure/memory"
	"minimart/pkg/metrics"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestService_UpdateRate(t *testing.T) {
	ctx := t.Context()
	repo := memory.NewRepository()
	svc := NewService(repo, nil, nil, metrics.Noop{}, time.Hour)

	r, err := domain.NewShippingRate("zone1", "method1", 599, 5000, true)
	require.NoError(t, err)
	err = repo.CreateRate(ctx, r)
	require.NoError(t, err)

	updated, err := svc.UpdateRate(ctx, r.ID, UpdateRateRequest{
		BaseCostCents:      1299,
		FreeThresholdCents: 10000,
		Active:             false,
	})
	require.NoError(t, err)
	assert.Equal(t, int64(1299), updated.BaseCostCents)
	assert.Equal(t, int64(10000), updated.FreeThresholdCents)
	assert.False(t, updated.Active)
}

func TestService_UpdateRate_NotFound(t *testing.T) {
	ctx := t.Context()
	repo := memory.NewRepository()
	svc := NewService(repo, nil, nil, metrics.Noop{}, time.Hour)

	_, err := svc.UpdateRate(ctx, "nonexistent", UpdateRateRequest{
		BaseCostCents:      1299,
		FreeThresholdCents: 10000,
		Active:             true,
	})
	require.ErrorIs(t, err, domain.ErrRateNotFound)
}
