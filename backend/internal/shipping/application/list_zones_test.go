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

func TestService_ListZones(t *testing.T) {
	ctx := t.Context()
	repo := memory.NewRepository()
	svc := NewService(repo, nil, nil, metrics.Noop{}, time.Hour)

	err := repo.CreateZone(ctx, domain.ShippingZone{ID: "z1", Name: "Brazil", Country: "BR"})
	require.NoError(t, err)
	err = repo.CreateZone(ctx, domain.ShippingZone{ID: "z2", Name: "SP", Country: "BR", State: "SP"})
	require.NoError(t, err)

	zones, err := svc.ListZones(ctx)
	require.NoError(t, err)
	assert.Len(t, zones, 2)
}
