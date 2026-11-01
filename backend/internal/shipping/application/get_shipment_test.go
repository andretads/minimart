package application

import (
	"minimart/internal/shipping/domain"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetShipment(t *testing.T) {
	t.Run("Shipment does not exist", func(t *testing.T) {
		svc, _, _ := newFeatureService(t)

		_, err := svc.GetShipment(t.Context(), "missing")
		require.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrShipmentNotFound)
	})

	t.Run("The shipment is returned", func(t *testing.T) {
		svc, repo, _ := newFeatureService(t)
		s := seedShipmentFeature(t, repo, "u1", "o1", domain.ShipmentStatusPending)

		got, err := svc.GetShipment(t.Context(), s.ID)
		require.NoError(t, err)
		assert.Equal(t, s.ID, got.ID)
		assert.Equal(t, "o1", got.OrderID)
	})
}
