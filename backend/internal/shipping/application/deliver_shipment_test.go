package application

import (
	"minimart/internal/shipping/domain"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDeliverShipment(t *testing.T) {
	t.Run("Shipment does not exist", func(t *testing.T) {
		svc, _, _ := newFeatureService(t)

		_, err := svc.DeliverShipment(t.Context(), "missing")
		require.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrShipmentNotFound)
	})

	t.Run("Invalid transition is rejected", func(t *testing.T) {
		svc, repo, _ := newFeatureService(t)
		s := seedShipmentFeature(t, repo, "u1", "o1", domain.ShipmentStatusProcessing)

		_, err := svc.DeliverShipment(t.Context(), s.ID)
		require.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrInvalidTransition)
	})

	t.Run("Shipment got delivered", func(t *testing.T) {
		svc, repo, _ := newFeatureService(t)
		s := seedShipmentFeature(t, repo, "u1", "o1", domain.ShipmentStatusShipped)

		_, err := svc.DeliverShipment(t.Context(), s.ID)
		require.NoError(t, err)

		stored, err := repo.GetShipment(t.Context(), s.ID)
		require.NoError(t, err)
		assert.Equal(t, domain.ShipmentStatusDelivered, stored.Status)
	})
}
