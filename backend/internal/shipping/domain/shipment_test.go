package domain

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewShipment(t *testing.T) {
	dest := DestinationSnapshot{FullName: "John", Street: "123 St", City: "City", Zip: "12345", Country: "BR"}

	tests := []struct {
		name      string
		orderID   string
		userID    string
		methodID  string
		dest      DestinationSnapshot
		cost      int64
		expectErr bool
		errIs     error
	}{
		{"valid", "order1", "user1", "method1", dest, 599, false, nil},
		{"empty order", "", "user1", "method1", dest, 599, true, ErrInvalidInput},
		{"empty user", "order1", "", "method1", dest, 599, true, ErrInvalidInput},
		{"empty method", "order1", "user1", "", dest, 599, true, ErrInvalidInput},
		{"negative cost", "order1", "user1", "method1", dest, -1, true, ErrInvalidTransition},
		{"invalid dest", "order1", "user1", "method1", DestinationSnapshot{}, 599, true, ErrInvalidQuoteDestination},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s, err := NewShipment(tt.orderID, tt.userID, tt.methodID, tt.dest, tt.cost)
			if tt.expectErr {
				require.Error(t, err)
				if tt.errIs != nil {
					require.ErrorIs(t, err, tt.errIs)
				}
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.orderID, s.OrderID)
			assert.Equal(t, tt.userID, s.UserID)
			assert.Equal(t, tt.methodID, s.MethodID)
			assert.Equal(t, tt.cost, s.ShippingCostCents)
			assert.Equal(t, ShipmentStatusPending, s.Status)
			assert.Nil(t, s.ShippedAt)
			assert.Nil(t, s.DeliveredAt)
			assert.Empty(t, s.TrackingCode)
			assert.Empty(t, s.Carrier)
			assert.False(t, s.CreatedAt.IsZero())
			assert.Equal(t, s.CreatedAt, s.UpdatedAt)
		})
	}
}

func TestShipment_Process(t *testing.T) {
	dest := DestinationSnapshot{FullName: "John", Street: "123 St", City: "City", Zip: "12345", Country: "BR"}
	s, _ := NewShipment("order1", "user1", "method1", dest, 599)

	err := s.Process()
	require.NoError(t, err)
	assert.Equal(t, ShipmentStatusProcessing, s.Status)

	err = s.Process()
	require.ErrorIs(t, err, ErrInvalidTransition)
}

func TestShipment_Dispatch(t *testing.T) {
	dest := DestinationSnapshot{FullName: "John", Street: "123 St", City: "City", Zip: "12345", Country: "BR"}
	s, _ := NewShipment("order1", "user1", "method1", dest, 599)
	_ = s.Process()

	err := s.Dispatch("TRACK123", "Correios")
	require.NoError(t, err)
	assert.Equal(t, ShipmentStatusShipped, s.Status)
	assert.Equal(t, "TRACK123", s.TrackingCode)
	assert.Equal(t, "Correios", s.Carrier)
	assert.NotNil(t, s.ShippedAt)

	err = s.Dispatch("TRACK456", "Other")
	require.ErrorIs(t, err, ErrInvalidTransition)
}

func TestShipment_Deliver(t *testing.T) {
	dest := DestinationSnapshot{FullName: "John", Street: "123 St", City: "City", Zip: "12345", Country: "BR"}
	s, _ := NewShipment("order1", "user1", "method1", dest, 599)
	_ = s.Process()
	_ = s.Dispatch("TRACK123", "Correios")

	err := s.Deliver()
	require.NoError(t, err)
	assert.Equal(t, ShipmentStatusDelivered, s.Status)
	assert.NotNil(t, s.DeliveredAt)

	err = s.Deliver()
	require.ErrorIs(t, err, ErrInvalidTransition)
}

func TestShipment_Fail(t *testing.T) {
	dest := DestinationSnapshot{FullName: "John", Street: "123 St", City: "City", Zip: "12345", Country: "BR"}

	// Fail from processing
	s, _ := NewShipment("order1", "user1", "method1", dest, 599)
	_ = s.Process()
	err := s.Fail()
	require.NoError(t, err)
	assert.Equal(t, ShipmentStatusFailed, s.Status)

	// Fail from shipped
	s2, _ := NewShipment("order2", "user1", "method1", dest, 599)
	_ = s2.Process()
	_ = s2.Dispatch("TRACK", "Correios")
	err = s2.Fail()
	require.NoError(t, err)
	assert.Equal(t, ShipmentStatusFailed, s2.Status)

	// Cannot fail from pending
	s3, _ := NewShipment("order3", "user1", "method1", dest, 599)
	err = s3.Fail()
	require.ErrorIs(t, err, ErrInvalidTransition)

	// Cannot fail from delivered
	s4, _ := NewShipment("order4", "user1", "method1", dest, 599)
	_ = s4.Process()
	_ = s4.Dispatch("TRACK", "Correios")
	_ = s4.Deliver()
	err = s4.Fail()
	require.ErrorIs(t, err, ErrInvalidTransition)

	// Cannot fail from failed
	s5, _ := NewShipment("order5", "user1", "method1", dest, 599)
	_ = s5.Process()
	_ = s5.Fail()
	err = s5.Fail()
	require.ErrorIs(t, err, ErrInvalidTransition)
}

func TestShipment_CanTransitionTo(t *testing.T) {
	tests := []struct {
		from    ShipmentStatus
		to      ShipmentStatus
		allowed bool
	}{
		{ShipmentStatusPending, ShipmentStatusProcessing, true},
		{ShipmentStatusPending, ShipmentStatusShipped, false},
		{ShipmentStatusProcessing, ShipmentStatusShipped, true},
		{ShipmentStatusProcessing, ShipmentStatusFailed, true},
		{ShipmentStatusProcessing, ShipmentStatusDelivered, false},
		{ShipmentStatusShipped, ShipmentStatusDelivered, true},
		{ShipmentStatusShipped, ShipmentStatusFailed, true},
		{ShipmentStatusDelivered, ShipmentStatusFailed, false},
		{ShipmentStatusFailed, ShipmentStatusDelivered, false},
	}

	for _, tt := range tests {
		t.Run(string(tt.from)+"_to_"+string(tt.to), func(t *testing.T) {
			dest := DestinationSnapshot{FullName: "John", Street: "123 St", City: "City", Zip: "12345", Country: "BR"}
			s, _ := NewShipment("order1", "user1", "method1", dest, 599)
			s.Status = tt.from
			assert.Equal(t, tt.allowed, s.CanTransitionTo(tt.to))
		})
	}
}

func TestShipment_IsTerminal(t *testing.T) {
	dest := DestinationSnapshot{FullName: "John", Street: "123 St", City: "City", Zip: "12345", Country: "BR"}
	s, _ := NewShipment("order1", "user1", "method1", dest, 599)

	assert.False(t, s.IsTerminal())
	s.Status = ShipmentStatusDelivered
	assert.True(t, s.IsTerminal())
	s.Status = ShipmentStatusFailed
	assert.True(t, s.IsTerminal())
}
