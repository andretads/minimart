//go:build integration

package test

import (
	"minimart/internal/shipping/domain"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDeliverShipmentIntegration(t *testing.T) {
	svc, _, _, repo, _ := newFeatureServices(t)

	t.Run("Shipment does not exist", func(t *testing.T) {
		_, err := svc.DeliverShipment(t.Context(), "missing")
		require.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrShipmentNotFound)
	})

	t.Run("Invalid transition is rejected", func(t *testing.T) {
		s := seedStoredShipment(t, svc, repo, "u1-"+uniqueShippingID(), "o1-"+uniqueShippingID(), domain.ShipmentStatusProcessing)
		_, err := svc.DeliverShipment(t.Context(), s.ID)
		require.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrInvalidTransition)
	})

	t.Run("Shipment got delivered", func(t *testing.T) {
		s := seedStoredShipment(t, svc, repo, "u1-"+uniqueShippingID(), "o1-"+uniqueShippingID(), domain.ShipmentStatusShipped)
		_, err := svc.DeliverShipment(t.Context(), s.ID)
		require.NoError(t, err)
		stored, err := repo.GetShipment(t.Context(), s.ID)
		require.NoError(t, err)
		assert.Equal(t, domain.ShipmentStatusDelivered, stored.Status)
	})
}
