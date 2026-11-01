package http

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetShipment(t *testing.T) {
	t.Run("found returns the shipment", func(t *testing.T) {
		env := newTestEnv(t)
		shipment := env.seedShipment(t, "user-1", "order-1", "")

		rec := env.doAs(t, "user-1", http.MethodGet, "/v1/shipping/shipments/"+shipment.ID, "")
		require.Equal(t, http.StatusOK, rec.Code)

		envl := decodeEnvelope(t, rec)
		var got ShipmentResponse
		require.NoError(t, json.Unmarshal(envl.Data, &got))
		assert.Equal(t, shipment.ID, got.ID)
		assert.Equal(t, "order-1", got.OrderID)
		assert.Equal(t, "pending", got.Status)
		assert.Equal(t, int64(599), got.ShippingCostCents)
	})

	t.Run("another user's shipment returns 403", func(t *testing.T) {
		env := newTestEnv(t)
		shipment := env.seedShipment(t, "user-1", "order-1", "")

		rec := env.doAs(t, "user-2", http.MethodGet, "/v1/shipping/shipments/"+shipment.ID, "")
		require.Equal(t, http.StatusForbidden, rec.Code)
	})

	t.Run("missing shipment returns 404", func(t *testing.T) {
		env := newTestEnv(t)
		rec := env.doAs(t, "user-1", http.MethodGet, "/v1/shipping/shipments/missing", "")
		require.Equal(t, http.StatusNotFound, rec.Code)
	})

	t.Run("without a user returns 401", func(t *testing.T) {
		env := newTestEnv(t)
		rec := env.do(t, http.MethodGet, "/v1/shipping/shipments/missing", "")
		require.Equal(t, http.StatusUnauthorized, rec.Code)
	})
}
