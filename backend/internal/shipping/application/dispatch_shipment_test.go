package application

import (
	"minimart/internal/shipping/domain"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDispatchShipment(t *testing.T) {
	t.Run("Shipment does not exist", func(t *testing.T) {
		svc, _, _ := newFeatureService(t)

		_, err := svc.DispatchShipment(t.Context(), "missing", DispatchShipmentRequest{TrackingCode: "BR123", Carrier: "Correios"})
		require.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrShipmentNotFound)
	})

	t.Run("Invalid transition is rejected", func(t *testing.T) {
		svc, repo, _ := newFeatureService(t)
		s := seedShipmentFeature(t, repo, "u1", "o1", domain.ShipmentStatusPending)

		_, err := svc.DispatchShipment(t.Context(), s.ID, DispatchShipmentRequest{TrackingCode: "BR123", Carrier: "Correios"})
		require.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrInvalidTransition)
	})

	t.Run("Missing tracking code is rejected", func(t *testing.T) {
		svc, repo, _ := newFeatureService(t)
		s := seedShipmentFeature(t, repo, "u1", "o1", domain.ShipmentStatusProcessing)

		_, err := svc.DispatchShipment(t.Context(), s.ID, DispatchShipmentRequest{TrackingCode: "", Carrier: "Correios"})
		require.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrInvalidTransition)
	})

	t.Run("Missing carrier is rejected", func(t *testing.T) {
		svc, repo, _ := newFeatureService(t)
		s := seedShipmentFeature(t, repo, "u1", "o1", domain.ShipmentStatusProcessing)

		_, err := svc.DispatchShipment(t.Context(), s.ID, DispatchShipmentRequest{TrackingCode: "BR123", Carrier: ""})
		require.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrInvalidTransition)
	})

	t.Run("Shipment got dispatched", func(t *testing.T) {
		svc, repo, _ := newFeatureService(t)
		s := seedShipmentFeature(t, repo, "u1", "o1", domain.ShipmentStatusProcessing)

		_, err := svc.DispatchShipment(t.Context(), s.ID, DispatchShipmentRequest{TrackingCode: "BR123", Carrier: "Correios"})
		require.NoError(t, err)

		stored, err := repo.GetShipment(t.Context(), s.ID)
		require.NoError(t, err)
		assert.Equal(t, domain.ShipmentStatusShipped, stored.Status)
		assert.Equal(t, "BR123", stored.TrackingCode)
		assert.Equal(t, "Correios", stored.Carrier)
	})
}
