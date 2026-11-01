package http

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestProcessShipment(t *testing.T) {
	t.Run("pending shipment becomes processing", func(t *testing.T) {
		env := newTestEnv(t)
		shipment := env.seedShipment(t, "user-1", "order-1", "")

		rec := env.do(t, http.MethodPost, "/v1/shipping/shipments/"+shipment.ID+"/process", "")
		require.Equal(t, http.StatusOK, rec.Code)

		envl := decodeEnvelope(t, rec)
		var got ShipmentResponse
		require.NoError(t, json.Unmarshal(envl.Data, &got))
		assert.Equal(t, "processing", got.Status)
	})

	t.Run("missing shipment returns 404", func(t *testing.T) {
		env := newTestEnv(t)
		rec := env.do(t, http.MethodPost, "/v1/shipping/shipments/missing/process", "")
		require.Equal(t, http.StatusNotFound, rec.Code)
	})

	t.Run("invalid transition returns 400", func(t *testing.T) {
		env := newTestEnv(t)
		shipment := env.seedShipment(t, "user-1", "order-1", "processing")

		rec := env.do(t, http.MethodPost, "/v1/shipping/shipments/"+shipment.ID+"/process", "")
		require.Equal(t, http.StatusBadRequest, rec.Code)
	})
}
