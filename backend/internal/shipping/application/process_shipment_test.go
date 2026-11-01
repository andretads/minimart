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

func TestService_ProcessShipment(t *testing.T) {
	ctx := t.Context()
	repo := memory.NewRepository()
	svc := NewService(repo, nil, nil, metrics.Noop{}, time.Hour)

	dest := domain.DestinationSnapshot{FullName: "John", Street: "123 St", City: "City", Zip: "12345", Country: "BR"}
	shipment, err := domain.NewShipment("order1", "user1", "method1", dest, 599)
	require.NoError(t, err)
	err = repo.CreateShipment(ctx, shipment)
	require.NoError(t, err)

	updated, err := svc.ProcessShipment(ctx, shipment.ID)
	require.NoError(t, err)
	assert.Equal(t, domain.ShipmentStatusProcessing, updated.Status)
}

func TestService_ProcessShipment_NotFound(t *testing.T) {
	ctx := t.Context()
	repo := memory.NewRepository()
	svc := NewService(repo, nil, nil, metrics.Noop{}, time.Hour)

	_, err := svc.ProcessShipment(ctx, "nonexistent")
	require.ErrorIs(t, err, domain.ErrShipmentNotFound)
}

func TestService_ProcessShipment_InvalidTransition(t *testing.T) {
	ctx := t.Context()
	repo := memory.NewRepository()
	svc := NewService(repo, nil, nil, metrics.Noop{}, time.Hour)

	dest := domain.DestinationSnapshot{FullName: "John", Street: "123 St", City: "City", Zip: "12345", Country: "BR"}
	s, err := domain.NewShipment("order1", "user1", "method1", dest, 599)
	require.NoError(t, err)
	s.Status = domain.ShipmentStatusProcessing // Already processing
	err = repo.CreateShipment(ctx, s)
	require.NoError(t, err)

	_, err = svc.ProcessShipment(ctx, s.ID)
	require.ErrorIs(t, err, domain.ErrInvalidTransition)
}
