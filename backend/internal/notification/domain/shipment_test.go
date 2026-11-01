package domain

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestNewShipmentCreated covers the shipment-created notification. Free
// shipping (zero cost) is valid; a negative cost is not.
func TestNewShipmentCreated(t *testing.T) {
	tests := []struct {
		name              string
		id                string
		orderID           string
		userID            string
		shippingCostCents int64
		wantErr           error
	}{
		{"valid", "n-1", "o-1", "u-1", 599, nil},
		{"free shipping", "n-1", "o-1", "u-1", 0, nil},
		{"empty id", "", "o-1", "u-1", 599, ErrEmptyID},
		{"empty order id", "n-1", "", "u-1", 599, ErrEmptyOrderID},
		{"empty user id", "n-1", "o-1", "", 599, ErrEmptyUserID},
		{"negative cost", "n-1", "o-1", "u-1", -1, ErrNegativeShippingCost},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			n, err := NewShipmentCreated(tt.id, tt.orderID, tt.userID, tt.shippingCostCents)
			if tt.wantErr != nil {
				assert.ErrorIs(t, err, tt.wantErr)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, TypeShipmentCreated, n.Type)
			assert.Equal(t, tt.orderID, n.OrderID)
			assert.Equal(t, tt.userID, n.UserID)
			assert.Contains(t, n.Body, formatMoney(tt.shippingCostCents))
			assert.False(t, n.SentAt.IsZero())
		})
	}
}

// TestNewShipmentShipped covers the shipment-shipped notification and that the
// tracking details reach the body when present.
func TestNewShipmentShipped(t *testing.T) {
	tests := []struct {
		name         string
		id           string
		orderID      string
		userID       string
		trackingCode string
		carrier      string
		wantErr      error
	}{
		{"valid", "n-1", "o-1", "u-1", "TRACK123", "Correios", nil},
		{"without tracking details", "n-1", "o-1", "u-1", "", "", nil},
		{"empty id", "", "o-1", "u-1", "TRACK123", "Correios", ErrEmptyID},
		{"empty order id", "n-1", "", "u-1", "TRACK123", "Correios", ErrEmptyOrderID},
		{"empty user id", "n-1", "o-1", "", "TRACK123", "Correios", ErrEmptyUserID},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			n, err := NewShipmentShipped(tt.id, tt.orderID, tt.userID, tt.trackingCode, tt.carrier)
			if tt.wantErr != nil {
				assert.ErrorIs(t, err, tt.wantErr)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, TypeShipmentShipped, n.Type)
			assert.Equal(t, tt.orderID, n.OrderID)
			assert.Contains(t, n.Body, tt.orderID)
			if tt.trackingCode != "" {
				assert.Contains(t, n.Body, "TRACK123")
				assert.Contains(t, n.Body, "Correios")
			}
			assert.False(t, n.SentAt.IsZero())
		})
	}
}

// TestNewShipmentDelivered covers the shipment-delivered notification.
func TestNewShipmentDelivered(t *testing.T) {
	tests := []struct {
		name    string
		id      string
		orderID string
		userID  string
		wantErr error
	}{
		{"valid", "n-1", "o-1", "u-1", nil},
		{"empty id", "", "o-1", "u-1", ErrEmptyID},
		{"empty order id", "n-1", "", "u-1", ErrEmptyOrderID},
		{"empty user id", "n-1", "o-1", "", ErrEmptyUserID},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			n, err := NewShipmentDelivered(tt.id, tt.orderID, tt.userID)
			if tt.wantErr != nil {
				assert.ErrorIs(t, err, tt.wantErr)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, TypeShipmentDelivered, n.Type)
			assert.Equal(t, tt.orderID, n.OrderID)
			assert.Contains(t, n.Body, tt.orderID)
			assert.False(t, n.SentAt.IsZero())
		})
	}
}
