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

func TestService_GetRate(t *testing.T) {
	ctx := t.Context()
	repo := memory.NewRepository()
	svc := NewService(repo, nil, nil, metrics.Noop{}, time.Hour)

	r, err := domain.NewShippingRate("zone1", "method1", 599, 5000, true)
	require.NoError(t, err)
	err = repo.CreateRate(ctx, r)
	require.NoError(t, err)

	got, err := svc.GetRate(ctx, r.ID)
	require.NoError(t, err)
	assert.Equal(t, r.ID, got.ID)
	assert.Equal(t, r.BaseCostCents, got.BaseCostCents)
}

func TestService_GetRate_NotFound(t *testing.T) {
	ctx := t.Context()
	repo := memory.NewRepository()
	svc := NewService(repo, nil, nil, metrics.Noop{}, time.Hour)

	_, err := svc.GetRate(ctx, "nonexistent")
	require.ErrorIs(t, err, domain.ErrRateNotFound)
}
