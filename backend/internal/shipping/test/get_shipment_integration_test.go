//go:build integration

package test

import (
	"minimart/internal/shipping/domain"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetShipmentIntegration(t *testing.T) {
	shipSvc, _, _, repo, _ := newFeatureServices(t)

	t.Run("Shipment does not exist", func(t *testing.T) {
		_, err := shipSvc.GetShipment(t.Context(), "missing")
		require.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrShipmentNotFound)
	})

	t.Run("The shipment is returned", func(t *testing.T) {
		s := seedStoredShipment(t, shipSvc, repo, "u1-"+uniqueShippingID(), "o1-"+uniqueShippingID(), domain.ShipmentStatusPending)
		got, err := shipSvc.GetShipment(t.Context(), s.ID)
		require.NoError(t, err)
		assert.Equal(t, s.ID, got.ID)
		assert.Equal(t, s.OrderID, got.OrderID)
	})
}
