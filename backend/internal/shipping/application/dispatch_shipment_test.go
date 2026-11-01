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

func TestService_DispatchShipment(t *testing.T) {
	ctx := t.Context()
	repo := memory.NewRepository()
	svc := NewService(repo, nil, nil, metrics.Noop{}, time.Hour)

	dest := domain.DestinationSnapshot{FullName: "John", Street: "123 St", City: "City", Zip: "12345", Country: "BR"}
	s, err := domain.NewShipment("order1", "user1", "method1", dest, 599)
	require.NoError(t, err)
	err = s.Process()
	require.NoError(t, err)
	err = repo.CreateShipment(ctx, s)
	require.NoError(t, err)

	updated, err := svc.DispatchShipment(ctx, s.ID, DispatchShipmentRequest{
		TrackingCode: "TRACK123",
		Carrier:      "Correios",
	})
	require.NoError(t, err)
	assert.Equal(t, domain.ShipmentStatusShipped, updated.Status)
	assert.Equal(t, "TRACK123", updated.TrackingCode)
	assert.Equal(t, "Correios", updated.Carrier)
	assert.NotNil(t, updated.ShippedAt)
}

func TestService_DispatchShipment_NotFound(t *testing.T) {
	ctx := t.Context()
	repo := memory.NewRepository()
	svc := NewService(repo, nil, nil, metrics.Noop{}, time.Hour)

	_, err := svc.DispatchShipment(ctx, "nonexistent", DispatchShipmentRequest{
		TrackingCode: "TRACK123",
		Carrier:      "Correios",
	})
	require.ErrorIs(t, err, domain.ErrShipmentNotFound)
}

func TestService_DispatchShipment_InvalidTransition(t *testing.T) {
	ctx := t.Context()
	repo := memory.NewRepository()
	svc := NewService(repo, nil, nil, metrics.Noop{}, time.Hour)

	dest := domain.DestinationSnapshot{FullName: "John", Street: "123 St", City: "City", Zip: "12345", Country: "BR"}
	s, err := domain.NewShipment("order1", "user1", "method1", dest, 599)
	require.NoError(t, err)
	err = repo.CreateShipment(ctx, s)
	require.NoError(t, err)

	_, err = svc.DispatchShipment(ctx, s.ID, DispatchShipmentRequest{
		TrackingCode: "TRACK123",
		Carrier:      "Correios",
	})
	require.ErrorIs(t, err, domain.ErrInvalidTransition)
}

func TestService_DispatchShipment_EmptyTrackingCode(t *testing.T) {
	ctx := t.Context()
	repo := memory.NewRepository()
	svc := NewService(repo, nil, nil, metrics.Noop{}, time.Hour)

	dest := domain.DestinationSnapshot{FullName: "John", Street: "123 St", City: "City", Zip: "12345", Country: "BR"}
	s, err := domain.NewShipment("order1", "user1", "method1", dest, 599)
	require.NoError(t, err)
	err = s.Process()
	require.NoError(t, err)
	err = repo.CreateShipment(ctx, s)
	require.NoError(t, err)

	_, err = svc.DispatchShipment(ctx, s.ID, DispatchShipmentRequest{
		TrackingCode: "",
		Carrier:      "Correios",
	})
	require.ErrorIs(t, err, domain.ErrInvalidTransition)
}
