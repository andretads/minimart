package domain

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewTrackingEvent(t *testing.T) {
	tests := []struct {
		name        string
		shipmentID  string
		status      ShipmentStatus
		description string
		location    string
		occurredAt  time.Time
		expectErr   bool
		errIs       error
	}{
		{"valid", "ship1", ShipmentStatusShipped, "Dispatched", "Sao Paulo", time.Now(), false, nil},
		{"empty shipment", "", ShipmentStatusShipped, "Dispatched", "Sao Paulo", time.Now(), true, ErrShipmentNotFound},
		{"empty description", "ship1", ShipmentStatusShipped, "", "Sao Paulo", time.Now(), true, ErrInvalidTransition},
		{"with empty location", "ship1", ShipmentStatusDelivered, "Delivered", "", time.Now(), false, nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e, err := NewTrackingEvent(tt.shipmentID, tt.status, tt.description, tt.location, tt.occurredAt)
			if tt.expectErr {
				require.Error(t, err)
				if tt.errIs != nil {
					require.ErrorIs(t, err, tt.errIs)
				}
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.shipmentID, e.ShipmentID)
			assert.Equal(t, tt.status, e.Status)
			assert.Equal(t, tt.description, e.Description)
			assert.Equal(t, tt.location, e.Location)
			assert.Equal(t, tt.occurredAt.UTC(), e.OccurredAt)
			assert.False(t, e.CreatedAt.IsZero())
		})
	}
}
