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

func TestService_AddTrackingEvent(t *testing.T) {
	ctx := t.Context()
	repo := memory.NewRepository()
	svc := NewService(repo, nil, nil, metrics.Noop{}, time.Hour)

	dest := domain.DestinationSnapshot{FullName: "John", Street: "123 St", City: "City", Zip: "12345", Country: "BR"}
	s, err := domain.NewShipment("order1", "user1", "method1", dest, 599)
	require.NoError(t, err)
	err = repo.CreateShipment(ctx, s)
	require.NoError(t, err)

	event, err := svc.AddTrackingEvent(ctx, AddTrackingEventRequest{
		ShipmentID:  s.ID,
		Status:      domain.ShipmentStatusShipped,
		Description: "Dispatched from warehouse",
		Location:    "Sao Paulo",
		OccurredAt:  time.Now().UTC(),
	})
	require.NoError(t, err)
	assert.Equal(t, s.ID, event.ShipmentID)
	assert.Equal(t, domain.ShipmentStatusShipped, event.Status)
	assert.Equal(t, "Dispatched from warehouse", event.Description)
	assert.Equal(t, "Sao Paulo", event.Location)
}

func TestService_AddTrackingEvent_NotFound(t *testing.T) {
	ctx := t.Context()
	repo := memory.NewRepository()
	svc := NewService(repo, nil, nil, metrics.Noop{}, time.Hour)

	_, err := svc.AddTrackingEvent(ctx, AddTrackingEventRequest{
		ShipmentID:  "nonexistent",
		Status:      domain.ShipmentStatusShipped,
		Description: "Test",
		Location:    "Test",
		OccurredAt:  time.Now().UTC(),
	})
	require.ErrorIs(t, err, domain.ErrShipmentNotFound)
}
