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

func TestService_UpdateZone(t *testing.T) {
	ctx := t.Context()
	repo := memory.NewRepository()
	svc := NewService(repo, nil, nil, metrics.Noop{}, time.Hour)

	z, err := domain.NewShippingZone("Brazil", "BR", "SP", "", "", "")
	require.NoError(t, err)
	err = repo.CreateZone(ctx, z)
	require.NoError(t, err)

	updated, err := svc.UpdateZone(ctx, z.ID, UpdateZoneRequest{
		Name:            "Brazil Updated",
		Country:         "BR",
		State:           "RJ",
		City:            "Rio",
		PostalCodeStart: "20000-000",
		PostalCodeEnd:   "29999-999",
	})
	require.NoError(t, err)
	assert.Equal(t, "Brazil Updated", updated.Name)
	assert.Equal(t, "BR", updated.Country)
	assert.Equal(t, "RJ", updated.State)
	assert.Equal(t, "Rio", updated.City)
	assert.Equal(t, "20000-000", updated.PostalCodeStart)
	assert.Equal(t, "29999-999", updated.PostalCodeEnd)
}

func TestService_UpdateZone_NotFound(t *testing.T) {
	ctx := t.Context()
	repo := memory.NewRepository()
	svc := NewService(repo, nil, nil, metrics.Noop{}, time.Hour)

	_, err := svc.UpdateZone(ctx, "nonexistent", UpdateZoneRequest{
		Name:    "Test",
		Country: "BR",
	})
	require.ErrorIs(t, err, domain.ErrZoneNotFound)
}
