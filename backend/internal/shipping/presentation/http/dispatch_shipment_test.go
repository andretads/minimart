package http

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDispatchShipment(t *testing.T) {
	t.Run("processing shipment becomes shipped", func(t *testing.T) {
		env := newTestEnv(t)
		shipment := env.seedShipment(t, "user-1", "order-1", "processing")

		rec := env.do(t, http.MethodPost, "/v1/shipping/shipments/"+shipment.ID+"/dispatch",
			`{"tracking_code":"TRACK-1","carrier":"Correios"}`)
		require.Equal(t, http.StatusOK, rec.Code)

		envl := decodeEnvelope(t, rec)
		var got ShipmentResponse
		require.NoError(t, json.Unmarshal(envl.Data, &got))
		assert.Equal(t, "shipped", got.Status)
		assert.Equal(t, "TRACK-1", got.TrackingCode)
		assert.Equal(t, "Correios", got.Carrier)
		require.NotNil(t, got.ShippedAt)
	})

	t.Run("missing shipment returns 404", func(t *testing.T) {
		env := newTestEnv(t)
		rec := env.do(t, http.MethodPost, "/v1/shipping/shipments/missing/dispatch",
			`{"tracking_code":"TRACK-1","carrier":"Correios"}`)
		require.Equal(t, http.StatusNotFound, rec.Code)
	})

	t.Run("invalid transition returns 400", func(t *testing.T) {
		env := newTestEnv(t)
		shipment := env.seedShipment(t, "user-1", "order-1", "")

		rec := env.do(t, http.MethodPost, "/v1/shipping/shipments/"+shipment.ID+"/dispatch",
			`{"tracking_code":"TRACK-1","carrier":"Correios"}`)
		require.Equal(t, http.StatusBadRequest, rec.Code)
	})

	t.Run("empty tracking code returns 400", func(t *testing.T) {
		env := newTestEnv(t)
		shipment := env.seedShipment(t, "user-1", "order-1", "processing")

		rec := env.do(t, http.MethodPost, "/v1/shipping/shipments/"+shipment.ID+"/dispatch",
			`{"tracking_code":"","carrier":"Correios"}`)
		require.Equal(t, http.StatusBadRequest, rec.Code)
	})

	t.Run("invalid JSON returns 400", func(t *testing.T) {
		env := newTestEnv(t)
		shipment := env.seedShipment(t, "user-1", "order-1", "processing")

		rec := env.do(t, http.MethodPost, "/v1/shipping/shipments/"+shipment.ID+"/dispatch", `{`)
		require.Equal(t, http.StatusBadRequest, rec.Code)
	})
}
