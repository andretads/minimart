package application

import (
	"minimart/internal/shipping/domain"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAddTrackingEvent(t *testing.T) {
	t.Run("Shipment does not exist", func(t *testing.T) {
		svc, _, _ := newFeatureService(t)

		_, err := svc.AddTrackingEvent(t.Context(), AddTrackingEventRequest{ShipmentID: "missing", Status: domain.ShipmentStatusShipped, Description: "Objeto enviado", Location: "Sao Paulo"})
		require.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrShipmentNotFound)
	})

	t.Run("Blank description is rejected", func(t *testing.T) {
		svc, repo, _ := newFeatureService(t)
		s := seedShipmentFeature(t, repo, "u1", "o1", domain.ShipmentStatusPending)

		_, err := svc.AddTrackingEvent(t.Context(), AddTrackingEventRequest{ShipmentID: s.ID, Status: domain.ShipmentStatusShipped, Description: "", Location: "Sao Paulo"})
		require.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrInvalidTransition)
	})

	t.Run("Tracking event got created", func(t *testing.T) {
		svc, repo, _ := newFeatureService(t)
		s := seedShipmentFeature(t, repo, "u1", "o1", domain.ShipmentStatusPending)

		_, err := svc.AddTrackingEvent(t.Context(), AddTrackingEventRequest{ShipmentID: s.ID, Status: domain.ShipmentStatusShipped, Description: "Objeto enviado", Location: "Sao Paulo"})
		require.NoError(t, err)

		events, err := repo.GetTrackingEvents(t.Context(), s.ID)
		require.NoError(t, err)
		require.Len(t, events, 1)
		assert.Equal(t, s.ID, events[0].ShipmentID)
		assert.Equal(t, domain.ShipmentStatusShipped, events[0].Status)
		assert.Equal(t, "Objeto enviado", events[0].Description)
		assert.Equal(t, "Sao Paulo", events[0].Location)
	})
}
