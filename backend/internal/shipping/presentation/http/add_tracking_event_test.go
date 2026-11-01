package http

import (
	"encoding/json"
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAddTrackingEvent(t *testing.T) {
	eventBody := `{"shipment_id":"%s","status":"processing","description":"Picked up","location":"Sao Paulo","occurred_at":"2026-01-01T10:00:00Z"}`

	t.Run("valid request returns 201", func(t *testing.T) {
		env := newTestEnv(t)
		shipment := env.seedShipment(t, "user-1", "order-1", "")

		rec := env.do(t, http.MethodPost, "/v1/shipping/shipments/"+shipment.ID+"/events",
			fmt.Sprintf(eventBody, shipment.ID))
		require.Equal(t, http.StatusCreated, rec.Code)

		envl := decodeEnvelope(t, rec)
		require.True(t, envl.Success)
		var event TrackingEventResponse
		require.NoError(t, json.Unmarshal(envl.Data, &event))
		assert.NotEmpty(t, event.ID)
		assert.Equal(t, shipment.ID, event.ShipmentID)
		assert.Equal(t, "processing", event.Status)
		assert.Equal(t, "Picked up", event.Description)
		assert.Equal(t, "Sao Paulo", event.Location)
	})

	t.Run("invalid JSON returns 400", func(t *testing.T) {
		env := newTestEnv(t)
		shipment := env.seedShipment(t, "user-1", "order-1", "")

		rec := env.do(t, http.MethodPost, "/v1/shipping/shipments/"+shipment.ID+"/events", `{`)
		require.Equal(t, http.StatusBadRequest, rec.Code)
	})

	t.Run("invalid occurred_at returns 400", func(t *testing.T) {
		env := newTestEnv(t)
		shipment := env.seedShipment(t, "user-1", "order-1", "")

		rec := env.do(t, http.MethodPost, "/v1/shipping/shipments/"+shipment.ID+"/events",
			`{"status":"processing","description":"Picked up","occurred_at":"not-a-time"}`)
		require.Equal(t, http.StatusBadRequest, rec.Code)
	})

	t.Run("missing description returns 400", func(t *testing.T) {
		env := newTestEnv(t)
		shipment := env.seedShipment(t, "user-1", "order-1", "")

		rec := env.do(t, http.MethodPost, "/v1/shipping/shipments/"+shipment.ID+"/events",
			`{"status":"processing","description":"","occurred_at":"2026-01-01T10:00:00Z"}`)
		require.Equal(t, http.StatusBadRequest, rec.Code)
	})

	t.Run("missing shipment returns 404", func(t *testing.T) {
		env := newTestEnv(t)
		rec := env.do(t, http.MethodPost, "/v1/shipping/shipments/missing/events",
			`{"status":"processing","description":"Picked up","occurred_at":"2026-01-01T10:00:00Z"}`)
		require.Equal(t, http.StatusNotFound, rec.Code)
	})
}
