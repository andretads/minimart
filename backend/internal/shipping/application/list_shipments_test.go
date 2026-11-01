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

func TestService_ListShipments(t *testing.T) {
	ctx := t.Context()
	repo := memory.NewRepository()
	svc := NewService(repo, nil, nil, metrics.Noop{}, time.Hour)

	dest := domain.DestinationSnapshot{FullName: "John", Street: "123 St", City: "City", Zip: "12345", Country: "BR"}

	// Create shipments for user1
	s1, err := domain.NewShipment("order1", "user1", "method1", dest, 599)
	require.NoError(t, err)
	err = repo.CreateShipment(ctx, s1)
	require.NoError(t, err)

	s2, err := domain.NewShipment("order2", "user1", "method1", dest, 1299)
	require.NoError(t, err)
	err = repo.CreateShipment(ctx, s2)
	require.NoError(t, err)

	// Create shipment for user2
	s3, err := domain.NewShipment("order3", "user2", "method1", dest, 599)
	require.NoError(t, err)
	err = repo.CreateShipment(ctx, s3)
	require.NoError(t, err)

	// List for user1
	shipments, total, err := svc.ListShipments(ctx, "user1", 1, 20)
	require.NoError(t, err)
	assert.Len(t, shipments, 2)
	assert.Equal(t, 2, total)
	assert.Equal(t, "order2", shipments[0].OrderID) // Newest first
	assert.Equal(t, "order1", shipments[1].OrderID)

	// List for user2
	shipments, total, err = svc.ListShipments(ctx, "user2", 1, 20)
	require.NoError(t, err)
	assert.Len(t, shipments, 1)
	assert.Equal(t, 1, total)
	assert.Equal(t, "order3", shipments[0].OrderID)

	// Pagination
	shipments, total, err = svc.ListShipments(ctx, "user1", 1, 1)
	require.NoError(t, err)
	assert.Len(t, shipments, 1)
	assert.Equal(t, 2, total)

	shipments, total, err = svc.ListShipments(ctx, "user1", 2, 1)
	require.NoError(t, err)
	assert.Len(t, shipments, 1)
	assert.Equal(t, 2, total)
}

func TestService_ListShipments_Empty(t *testing.T) {
	ctx := t.Context()
	repo := memory.NewRepository()
	svc := NewService(repo, nil, nil, metrics.Noop{}, time.Hour)

	shipments, total, err := svc.ListShipments(ctx, "user1", 1, 20)
	require.NoError(t, err)
	assert.Empty(t, shipments)
	assert.Equal(t, 0, total)
}
