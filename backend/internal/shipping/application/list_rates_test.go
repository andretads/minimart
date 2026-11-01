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

func TestService_ListRates(t *testing.T) {
	ctx := t.Context()
	repo := memory.NewRepository()
	svc := NewService(repo, nil, nil, metrics.Noop{}, time.Hour)

	err := repo.CreateRate(ctx, domain.ShippingRate{ID: "r1", ZoneID: "z1", MethodID: "m1", BaseCostCents: 599, FreeThresholdCents: 5000, Active: true})
	require.NoError(t, err)
	err = repo.CreateRate(ctx, domain.ShippingRate{ID: "r2", ZoneID: "z1", MethodID: "m2", BaseCostCents: 1299, FreeThresholdCents: 10000, Active: true})
	require.NoError(t, err)
	err = repo.CreateRate(ctx, domain.ShippingRate{ID: "r3", ZoneID: "z2", MethodID: "m1", BaseCostCents: 999, FreeThresholdCents: 0, Active: true})
	require.NoError(t, err)

	rates, err := svc.ListRates(ctx, "z1", "")
	require.NoError(t, err)
	assert.Len(t, rates, 2)

	rates, err = svc.ListRates(ctx, "", "m1")
	require.NoError(t, err)
	assert.Len(t, rates, 2)

	rates, err = svc.ListRates(ctx, "z1", "m1")
	require.NoError(t, err)
	assert.Len(t, rates, 1)
}
