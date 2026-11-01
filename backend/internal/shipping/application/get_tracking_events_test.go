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

func TestService_GetTrackingEvents(t *testing.T) {
	ctx := t.Context()
	repo := memory.NewRepository()
	svc := NewService(repo, nil, nil, metrics.Noop{}, time.Hour)

	dest := domain.DestinationSnapshot{FullName: "John", Street: "123 St", City: "City", Zip: "12345", Country: "BR"}
	s, err := domain.NewShipment("order1", "user1", "method1", dest, 599)
	require.NoError(t, err)
	err = repo.CreateShipment(ctx, s)
	require.NoError(t, err)

	// Add tracking events
	e1, err := domain.NewTrackingEvent(s.ID, domain.ShipmentStatusShipped, "Dispatched", "Sao Paulo", time.Now().UTC())
	require.NoError(t, err)
	err = repo.AddTrackingEvent(ctx, e1)
	require.NoError(t, err)

	e2, err := domain.NewTrackingEvent(s.ID, domain.ShipmentStatusDelivered, "Delivered", "Recipient address", time.Now().UTC())
	require.NoError(t, err)
	err = repo.AddTrackingEvent(ctx, e2)
	require.NoError(t, err)

	// Get events
	events, err := svc.GetTrackingEvents(ctx, s.ID)
	require.NoError(t, err)
	assert.Len(t, events, 2)
	assert.Equal(t, domain.ShipmentStatusDelivered, events[0].Status) // Newest first
	assert.Equal(t, domain.ShipmentStatusShipped, events[1].Status)
}

func TestService_GetTrackingEvents_Empty(t *testing.T) {
	ctx := t.Context()
	repo := memory.NewRepository()
	svc := NewService(repo, nil, nil, metrics.Noop{}, time.Hour)

	dest := domain.DestinationSnapshot{FullName: "John", Street: "123 St", City: "City", Zip: "12345", Country: "BR"}
	s, err := domain.NewShipment("order1", "user1", "method1", dest, 599)
	require.NoError(t, err)
	err = repo.CreateShipment(ctx, s)
	require.NoError(t, err)

	events, err := svc.GetTrackingEvents(ctx, s.ID)
	require.NoError(t, err)
	assert.Empty(t, events)
}

func TestService_GetTrackingEvents_NotFound(t *testing.T) {
	ctx := t.Context()
	repo := memory.NewRepository()
	svc := NewService(repo, nil, nil, metrics.Noop{}, time.Hour)

	events, err := svc.GetTrackingEvents(ctx, "nonexistent")
	require.NoError(t, err)
	assert.Empty(t, events)
}
