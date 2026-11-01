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

func TestService_CreateRate(t *testing.T) {
	ctx := t.Context()
	repo := memory.NewRepository()
	svc := NewService(repo, nil, nil, metrics.Noop{}, time.Hour)

	rate, err := svc.CreateRate(ctx, CreateRateRequest{
		ZoneID:             "zone1",
		MethodID:           "method1",
		BaseCostCents:      599,
		FreeThresholdCents: 5000,
		Active:             true,
	})
	require.NoError(t, err)
	assert.Equal(t, int64(599), rate.BaseCostCents)
	assert.Equal(t, int64(5000), rate.FreeThresholdCents)
	assert.True(t, rate.Active)
	assert.NotEmpty(t, rate.ID)
}

func TestService_CreateRate_Invalid(t *testing.T) {
	ctx := t.Context()
	repo := memory.NewRepository()
	svc := NewService(repo, nil, nil, metrics.Noop{}, time.Hour)

	_, err := svc.CreateRate(ctx, CreateRateRequest{
		ZoneID:             "",
		MethodID:           "method1",
		BaseCostCents:      599,
		FreeThresholdCents: 5000,
		Active:             true,
	})
	require.ErrorIs(t, err, domain.ErrRateNotFound)
}
