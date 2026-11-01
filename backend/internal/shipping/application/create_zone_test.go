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

func TestService_CreateZone(t *testing.T) {
	ctx := t.Context()
	repo := memory.NewRepository()
	svc := NewService(repo, nil, nil, metrics.Noop{}, time.Hour)

	zone, err := svc.CreateZone(ctx, CreateZoneRequest{
		Name:            "Brazil",
		Country:         "BR",
		State:           "SP",
		City:            "Sao Paulo",
		PostalCodeStart: "01000-000",
		PostalCodeEnd:   "09999-999",
	})
	require.NoError(t, err)
	assert.Equal(t, "Brazil", zone.Name)
	assert.Equal(t, "BR", zone.Country)
	assert.Equal(t, "SP", zone.State)
	assert.Equal(t, "Sao Paulo", zone.City)
	assert.Equal(t, "01000-000", zone.PostalCodeStart)
	assert.Equal(t, "09999-999", zone.PostalCodeEnd)
	assert.NotEmpty(t, zone.ID)
}

func TestService_CreateZone_Invalid(t *testing.T) {
	ctx := t.Context()
	repo := memory.NewRepository()
	svc := NewService(repo, nil, nil, metrics.Noop{}, time.Hour)

	_, err := svc.CreateZone(ctx, CreateZoneRequest{
		Name:    "",
		Country: "BR",
	})
	require.ErrorIs(t, err, domain.ErrZoneNotFound)
}
