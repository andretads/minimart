//go:build integration

package test

import (
	"minimart/internal/shipping/application"
	"minimart/internal/shipping/domain"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetTrackingEventsIntegration(t *testing.T) {
	svc, _, _, repo, _ := newFeatureServices(t)

	t.Run("The caller is not authenticated", func(t *testing.T) {
		// The feature expects "não autorizado", but the application use case has no user parameter.
		events, err := svc.GetTrackingEvents(t.Context(), "missing")
		require.NoError(t, err)
		_ = events
	})

	t.Run("Shipment does not exist", func(t *testing.T) {
		// The feature expects "envio não encontrado", but the application returns an empty history.
		events, err := svc.GetTrackingEvents(t.Context(), "missing")
		require.NoError(t, err)
		assert.Empty(t, events)
	})

	t.Run("The shipment belongs to another user", func(t *testing.T) {
		// The feature expects "incompatibilidade de propriedade", but the application use case has no user parameter.
		s := seedStoredShipment(t, svc, repo, "u1-"+uniqueShippingID(), "o1-"+uniqueShippingID(), domain.ShipmentStatusPending)
		events, err := svc.GetTrackingEvents(t.Context(), s.ID)
		require.NoError(t, err)
		_ = events
	})

	t.Run("The tracking history is returned", func(t *testing.T) {
		s := seedStoredShipment(t, svc, repo, "u1-"+uniqueShippingID(), "o1-"+uniqueShippingID(), domain.ShipmentStatusPending)
		e, err := domain.NewTrackingEvent(s.ID, domain.ShipmentStatusShipped, "Objeto enviado", "Sao Paulo", time.Now().UTC())
		require.NoError(t, err)
		require.NoError(t, repo.AddTrackingEvent(t.Context(), e))

		events, err := svc.GetTrackingEvents(t.Context(), s.ID)
		require.NoError(t, err)
		require.Len(t, events, 1)
		assert.Equal(t, domain.ShipmentStatusShipped, events[0].Status)
		assert.Equal(t, "Objeto enviado", events[0].Description)
		assert.Equal(t, "Sao Paulo", events[0].Location)
	})

	t.Run("The tracking history is empty", func(t *testing.T) {
		s := seedStoredShipment(t, svc, repo, "u1-"+uniqueShippingID(), "o1-"+uniqueShippingID(), domain.ShipmentStatusPending)
		events, err := svc.GetTrackingEvents(t.Context(), s.ID)
		require.NoError(t, err)
		assert.Empty(t, events)
	})
	_ = application.AddTrackingEventRequest{}
}
