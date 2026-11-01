//go:build integration

package test

import (
	"minimart/internal/shipping/application"
	"minimart/internal/shipping/domain"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDispatchShipmentIntegration(t *testing.T) {
	svc, _, _, repo, _ := newFeatureServices(t)

	t.Run("Shipment does not exist", func(t *testing.T) {
		_, err := svc.DispatchShipment(t.Context(), "missing", application.DispatchShipmentRequest{TrackingCode: "BR123", Carrier: "Correios"})
		require.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrShipmentNotFound)
	})

	t.Run("Invalid transition is rejected", func(t *testing.T) {
		s := seedStoredShipment(t, svc, repo, "u1-"+uniqueShippingID(), "o1-"+uniqueShippingID(), domain.ShipmentStatusPending)
		_, err := svc.DispatchShipment(t.Context(), s.ID, application.DispatchShipmentRequest{TrackingCode: "BR123", Carrier: "Correios"})
		require.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrInvalidTransition)
	})

	t.Run("Missing tracking code is rejected", func(t *testing.T) {
		s := seedStoredShipment(t, svc, repo, "u1-"+uniqueShippingID(), "o1-"+uniqueShippingID(), domain.ShipmentStatusProcessing)
		_, err := svc.DispatchShipment(t.Context(), s.ID, application.DispatchShipmentRequest{TrackingCode: "", Carrier: "Correios"})
		require.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrInvalidTransition)
	})

	t.Run("Missing carrier is rejected", func(t *testing.T) {
		s := seedStoredShipment(t, svc, repo, "u1-"+uniqueShippingID(), "o1-"+uniqueShippingID(), domain.ShipmentStatusProcessing)
		_, err := svc.DispatchShipment(t.Context(), s.ID, application.DispatchShipmentRequest{TrackingCode: "BR123", Carrier: ""})
		require.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrInvalidTransition)
	})

	t.Run("Shipment got dispatched", func(t *testing.T) {
		s := seedStoredShipment(t, svc, repo, "u1-"+uniqueShippingID(), "o1-"+uniqueShippingID(), domain.ShipmentStatusProcessing)
		_, err := svc.DispatchShipment(t.Context(), s.ID, application.DispatchShipmentRequest{TrackingCode: "BR123", Carrier: "Correios"})
		require.NoError(t, err)
		stored, err := repo.GetShipment(t.Context(), s.ID)
		require.NoError(t, err)
		assert.Equal(t, domain.ShipmentStatusShipped, stored.Status)
		assert.Equal(t, "BR123", stored.TrackingCode)
		assert.Equal(t, "Correios", stored.Carrier)
	})
}
