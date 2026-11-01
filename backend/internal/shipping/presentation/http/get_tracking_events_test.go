package http

import (
	"encoding/json"
	"minimart/internal/shipping/domain"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetTrackingEvents(t *testing.T) {
	t.Run("returns the shipment tracking history", func(t *testing.T) {
		env := newTestEnv(t)
		shipment := env.seedShipment(t, "user-1", "order-1", "")
		event, err := domain.NewTrackingEvent(shipment.ID, domain.ShipmentStatusShipped, "Enviado", "Sao Paulo", time.Now().UTC())
		require.NoError(t, err)
		require.NoError(t, env.repo.AddTrackingEvent(t.Context(), event))

		rec := env.doAs(t, "user-1", http.MethodGet, "/v1/shipping/shipments/"+shipment.ID+"/events", "")
		require.Equal(t, http.StatusOK, rec.Code)

		envl := decodeEnvelope(t, rec)
		var got []TrackingEventResponse
		require.NoError(t, json.Unmarshal(envl.Data, &got))
		require.Len(t, got, 1)
		assert.Equal(t, event.ID, got[0].ID)
		assert.Equal(t, shipment.ID, got[0].ShipmentID)
		assert.Equal(t, "Enviado", got[0].Description)
		assert.Equal(t, "Sao Paulo", got[0].Location)
	})

	t.Run("a shipment of another user is not readable", func(t *testing.T) {
		env := newTestEnv(t)
		shipment := env.seedShipment(t, "user-1", "order-1", "")

		rec := env.doAs(t, "user-2", http.MethodGet, "/v1/shipping/shipments/"+shipment.ID+"/events", "")
		require.Equal(t, http.StatusForbidden, rec.Code)
	})

	t.Run("missing shipment returns 404", func(t *testing.T) {
		env := newTestEnv(t)
		rec := env.doAs(t, "user-1", http.MethodGet, "/v1/shipping/shipments/missing/events", "")
		require.Equal(t, http.StatusNotFound, rec.Code)
	})

	t.Run("without a user returns 401", func(t *testing.T) {
		env := newTestEnv(t)
		rec := env.do(t, http.MethodGet, "/v1/shipping/shipments/missing/events", "")
		require.Equal(t, http.StatusUnauthorized, rec.Code)
	})
}
