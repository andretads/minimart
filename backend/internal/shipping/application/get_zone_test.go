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

func TestService_GetZone(t *testing.T) {
	ctx := t.Context()
	repo := memory.NewRepository()
	svc := NewService(repo, nil, nil, metrics.Noop{}, time.Hour)

	z, err := domain.NewShippingZone("Brazil", "BR", "SP", "", "", "")
	require.NoError(t, err)
	err = repo.CreateZone(ctx, z)
	require.NoError(t, err)

	got, err := svc.GetZone(ctx, z.ID)
	require.NoError(t, err)
	assert.Equal(t, z.ID, got.ID)
	assert.Equal(t, z.Name, got.Name)
}

func TestService_GetZone_NotFound(t *testing.T) {
	ctx := t.Context()
	repo := memory.NewRepository()
	svc := NewService(repo, nil, nil, metrics.Noop{}, time.Hour)

	_, err := svc.GetZone(ctx, "nonexistent")
	require.ErrorIs(t, err, domain.ErrZoneNotFound)
}
