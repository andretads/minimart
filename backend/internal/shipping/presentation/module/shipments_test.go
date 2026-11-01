package module

import (
	"minimart/internal/shipping/application"
	"minimart/internal/shipping/domain"
	"minimart/internal/shipping/infrastructure/memory"
	shippingclient "minimart/internal/shipping/presentation/module/client"
	"minimart/pkg/metrics"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func createTestShipment(t *testing.T, repo *memory.Repository) domain.Shipment {
	t.Helper()
	zone, _ := domain.NewShippingZone("Brazil", "BR", "", "", "", "")
	method, _ := domain.NewShippingMethod("Standard", "Standard delivery", true, 3, 5)
	require.NoError(t, repo.CreateZone(t.Context(), zone))
	require.NoError(t, repo.CreateMethod(t.Context(), method))

	addr := domain.DestinationSnapshot{
		Label:    "Home",
		FullName: "John Doe",
		Street:   "123 Main St",
		City:     "Sao Paulo",
		State:    "SP",
		Zip:      "01000-000",
		Country:  "BR",
		Phone:    "+5511999999999",
	}
	sh, _ := domain.NewShipment("order-1", "user-1", method.ID, addr, 599)
	require.NoError(t, repo.CreateShipment(t.Context(), sh))
	return sh
}

func TestGetShipment(t *testing.T) {
	repo := memory.NewRepository()
	sh := createTestShipment(t, repo)

	t.Run("returns shipment mapped to client type", func(t *testing.T) {
		adapter := New(application.NewService(repo, testModules{}, nil, metrics.Noop{}, time.Hour))

		got, err := adapter.GetShipment(t.Context(), sh.ID)
		require.NoError(t, err)
		assert.Equal(t, sh.ID, got.ID)
		assert.Equal(t, "order-1", got.OrderID)
		assert.Equal(t, "user-1", got.UserID)
		assert.Equal(t, sh.MethodID, got.MethodID)
		assert.Empty(t, got.TrackingCode) // Not set until Dispatch
		assert.Empty(t, got.Carrier)      // Not set until Dispatch
		assert.Equal(t, int64(599), got.ShippingCostCents)
		assert.Equal(t, string(domain.ShipmentStatusPending), got.Status)
		assert.Nil(t, got.ShippedAt)
		assert.Nil(t, got.DeliveredAt)
		assert.NotEmpty(t, got.CreatedAt)
		assert.NotEmpty(t, got.UpdatedAt)
		assert.Equal(t, "Home", got.DestinationSnapshot.Label)
		assert.Equal(t, "John Doe", got.DestinationSnapshot.FullName)
	})

	t.Run("returns not-found error for missing shipment", func(t *testing.T) {
		adapter := New(application.NewService(memory.NewRepository(), testModules{}, nil, metrics.Noop{}, time.Hour))

		_, err := adapter.GetShipment(t.Context(), "missing")
		assert.ErrorIs(t, err, shippingclient.ErrShipmentNotFound)
	})
}

func TestGetShipmentByOrderID(t *testing.T) {
	repo := memory.NewRepository()
	sh := createTestShipment(t, repo)

	t.Run("returns shipment by order ID mapped to client type", func(t *testing.T) {
		adapter := New(application.NewService(repo, testModules{}, nil, metrics.Noop{}, time.Hour))

		got, err := adapter.GetShipmentByOrderID(t.Context(), "order-1")
		require.NoError(t, err)
		assert.Equal(t, sh.ID, got.ID)
		assert.Equal(t, "order-1", got.OrderID)
	})

	t.Run("returns not-found error for missing order ID", func(t *testing.T) {
		adapter := New(application.NewService(memory.NewRepository(), testModules{}, nil, metrics.Noop{}, time.Hour))

		_, err := adapter.GetShipmentByOrderID(t.Context(), "missing")
		assert.ErrorIs(t, err, shippingclient.ErrShipmentNotFound)
	})
}

func TestListShipments(t *testing.T) {
	repo := memory.NewRepository()
	zone, _ := domain.NewShippingZone("Brazil", "BR", "", "", "", "")
	method, _ := domain.NewShippingMethod("Standard", "Standard delivery", true, 3, 5)
	require.NoError(t, repo.CreateZone(t.Context(), zone))
	require.NoError(t, repo.CreateMethod(t.Context(), method))

	addr := domain.DestinationSnapshot{
		Label:    "Home",
		FullName: "John Doe",
		Street:   "123 Main St",
		City:     "Sao Paulo",
		State:    "SP",
		Zip:      "01000-000",
		Country:  "BR",
		Phone:    "+5511999999999",
	}
	sh1, _ := domain.NewShipment("order-1", "user-1", method.ID, addr, 599)
	sh2, _ := domain.NewShipment("order-2", "user-1", method.ID, addr, 599)
	sh3, _ := domain.NewShipment("order-3", "user-2", method.ID, addr, 599)
	require.NoError(t, repo.CreateShipment(t.Context(), sh1))
	require.NoError(t, repo.CreateShipment(t.Context(), sh2))
	require.NoError(t, repo.CreateShipment(t.Context(), sh3))

	t.Run("returns paginated shipments for user", func(t *testing.T) {
		adapter := New(application.NewService(repo, testModules{}, nil, metrics.Noop{}, time.Hour))

		got, total, err := adapter.ListShipments(t.Context(), "user-1", 1, 10)
		require.NoError(t, err)
		assert.Equal(t, 2, total)
		require.Len(t, got, 2)

		// Check both orders are present without assuming order
		orderIDs := make(map[string]bool)
		for _, s := range got {
			orderIDs[s.OrderID] = true
		}
		assert.True(t, orderIDs["order-1"])
		assert.True(t, orderIDs["order-2"])
	})

	t.Run("returns empty for user with no shipments", func(t *testing.T) {
		adapter := New(application.NewService(repo, testModules{}, nil, metrics.Noop{}, time.Hour))

		got, total, err := adapter.ListShipments(t.Context(), "user-999", 1, 10)
		require.NoError(t, err)
		assert.Equal(t, 0, total)
		assert.Empty(t, got)
	})

	t.Run("respects pagination", func(t *testing.T) {
		adapter := New(application.NewService(repo, testModules{}, nil, metrics.Noop{}, time.Hour))

		got, total, err := adapter.ListShipments(t.Context(), "user-1", 1, 1)
		require.NoError(t, err)
		assert.Equal(t, 2, total)
		require.Len(t, got, 1)
	})
}

func TestGetTrackingEvents(t *testing.T) {
	repo := memory.NewRepository()
	sh := createTestShipment(t, repo)

	// Add tracking events
	event1 := domain.TrackingEvent{
		ID:          "evt-1",
		ShipmentID:  sh.ID,
		Status:      domain.ShipmentStatusProcessing,
		Description: "Package in transit",
		Location:    "Sao Paulo",
		OccurredAt:  time.Now().Add(-2 * time.Hour),
	}
	event2 := domain.TrackingEvent{
		ID:          "evt-2",
		ShipmentID:  sh.ID,
		Status:      domain.ShipmentStatusShipped,
		Description: "Out for delivery",
		Location:    "Sao Paulo",
		OccurredAt:  time.Now().Add(-1 * time.Hour),
	}
	require.NoError(t, repo.AddTrackingEvent(t.Context(), event1))
	require.NoError(t, repo.AddTrackingEvent(t.Context(), event2))

	t.Run("returns tracking events mapped to client type", func(t *testing.T) {
		adapter := New(application.NewService(repo, testModules{}, nil, metrics.Noop{}, time.Hour))

		got, err := adapter.GetTrackingEvents(t.Context(), sh.ID)
		require.NoError(t, err)
		require.Len(t, got, 2)
		// Events returned in reverse chronological order (newest first)
		assert.Equal(t, "evt-2", got[0].ID)
		assert.Equal(t, string(domain.ShipmentStatusShipped), got[0].Status)
		assert.Equal(t, "evt-1", got[1].ID)
		assert.Equal(t, string(domain.ShipmentStatusProcessing), got[1].Status)
	})

	t.Run("returns empty for shipment with no events", func(t *testing.T) {
		repo2 := memory.NewRepository()
		sh2 := createTestShipment(t, repo2)
		adapter := New(application.NewService(repo2, testModules{}, nil, metrics.Noop{}, time.Hour))

		got, err := adapter.GetTrackingEvents(t.Context(), sh2.ID)
		require.NoError(t, err)
		assert.Empty(t, got)
	})
}
