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

func TestService_GetShipment(t *testing.T) {
	ctx := t.Context()
	repo := memory.NewRepository()
	svc := NewService(repo, nil, nil, metrics.Noop{}, time.Hour)

	dest := domain.DestinationSnapshot{FullName: "John", Street: "123 St", City: "City", Zip: "12345", Country: "BR"}
	shipment, err := domain.NewShipment("order1", "user1", "method1", dest, 599)
	require.NoError(t, err)
	err = repo.CreateShipment(ctx, shipment)
	require.NoError(t, err)

	got, err := svc.GetShipment(ctx, shipment.ID)
	require.NoError(t, err)
	assert.Equal(t, shipment.ID, got.ID)
	assert.Equal(t, shipment.OrderID, got.OrderID)
}

func TestService_GetShipment_NotFound(t *testing.T) {
	ctx := t.Context()
	repo := memory.NewRepository()
	svc := NewService(repo, nil, nil, metrics.Noop{}, time.Hour)

	_, err := svc.GetShipment(ctx, "nonexistent")
	require.ErrorIs(t, err, domain.ErrShipmentNotFound)
}

func TestService_GetShipmentByOrderID(t *testing.T) {
	ctx := t.Context()
	repo := memory.NewRepository()
	svc := NewService(repo, nil, nil, metrics.Noop{}, time.Hour)

	dest := domain.DestinationSnapshot{FullName: "John", Street: "123 St", City: "City", Zip: "12345", Country: "BR"}
	shipment, err := domain.NewShipment("order1", "user1", "method1", dest, 599)
	require.NoError(t, err)
	err = repo.CreateShipment(ctx, shipment)
	require.NoError(t, err)

	got, err := svc.GetShipmentByOrderID(ctx, "order1")
	require.NoError(t, err)
	assert.Equal(t, shipment.ID, got.ID)
}

func TestService_GetShipmentByOrderID_NotFound(t *testing.T) {
	ctx := t.Context()
	repo := memory.NewRepository()
	svc := NewService(repo, nil, nil, metrics.Noop{}, time.Hour)

	_, err := svc.GetShipmentByOrderID(ctx, "nonexistent")
	require.ErrorIs(t, err, domain.ErrShipmentNotFound)
}
