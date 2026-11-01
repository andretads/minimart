package http

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestListShipments(t *testing.T) {
	t.Run("lists only the authenticated user's shipments", func(t *testing.T) {
		env := newTestEnv(t)
		wanted := env.seedShipment(t, "user-1", "order-1", "")
		env.seedShipment(t, "user-1", "order-2", "")
		env.seedShipment(t, "user-2", "order-3", "")

		rec := env.doAs(t, "user-1", http.MethodGet, "/v1/shipping/shipments", "")
		require.Equal(t, http.StatusOK, rec.Code)

		envl := decodeEnvelope(t, rec)
		var shipments []ShipmentResponse
		require.NoError(t, json.Unmarshal(envl.Data, &shipments))
		require.Len(t, shipments, 2)
		require.NotNil(t, envl.Meta)
		assert.Equal(t, 2, envl.Meta.Total)
		assert.Contains(t, shipmentIDs(shipments), wanted.ID)
	})

	t.Run("defaults to the authenticated user", func(t *testing.T) {
		env := newTestEnv(t)
		env.seedShipment(t, "user-1", "order-1", "")
		env.seedShipment(t, "user-2", "order-2", "")

		rec := env.doAs(t, "user-1", http.MethodGet, "/v1/shipping/shipments", "")
		require.Equal(t, http.StatusOK, rec.Code)

		envl := decodeEnvelope(t, rec)
		var shipments []ShipmentResponse
		require.NoError(t, json.Unmarshal(envl.Data, &shipments))
		assert.Len(t, shipments, 1)
		assert.Equal(t, "user-1", shipments[0].UserID)
	})

	t.Run("filters by order_id", func(t *testing.T) {
		env := newTestEnv(t)
		shipment := env.seedShipment(t, "user-1", "order-1", "")
		env.seedShipment(t, "user-1", "order-2", "")

		rec := env.doAs(t, "user-1", http.MethodGet, "/v1/shipping/shipments?order_id=order-1", "")
		require.Equal(t, http.StatusOK, rec.Code)

		envl := decodeEnvelope(t, rec)
		var shipments []ShipmentResponse
		require.NoError(t, json.Unmarshal(envl.Data, &shipments))
		require.Len(t, shipments, 1)
		assert.Equal(t, shipment.ID, shipments[0].ID)
		require.NotNil(t, envl.Meta)
		assert.Equal(t, 1, envl.Meta.Total)
	})

	t.Run("unknown order_id returns an empty page", func(t *testing.T) {
		env := newTestEnv(t)
		env.seedShipment(t, "user-1", "order-1", "")

		rec := env.doAs(t, "user-1", http.MethodGet, "/v1/shipping/shipments?order_id=missing", "")
		require.Equal(t, http.StatusOK, rec.Code)

		envl := decodeEnvelope(t, rec)
		var shipments []ShipmentResponse
		require.NoError(t, json.Unmarshal(envl.Data, &shipments))
		assert.Empty(t, shipments)
		require.NotNil(t, envl.Meta)
		assert.Equal(t, 0, envl.Meta.Total)
		assert.Equal(t, 0, envl.Meta.TotalPages)
	})

	t.Run("another user's order_id returns an empty page", func(t *testing.T) {
		env := newTestEnv(t)
		env.seedShipment(t, "user-1", "order-1", "")

		rec := env.doAs(t, "user-2", http.MethodGet, "/v1/shipping/shipments?order_id=order-1", "")
		require.Equal(t, http.StatusOK, rec.Code)

		envl := decodeEnvelope(t, rec)
		var shipments []ShipmentResponse
		require.NoError(t, json.Unmarshal(envl.Data, &shipments))
		assert.Empty(t, shipments)
	})

	t.Run("missing user returns 401", func(t *testing.T) {
		env := newTestEnv(t)
		rec := env.do(t, http.MethodGet, "/v1/shipping/shipments", "")
		require.Equal(t, http.StatusUnauthorized, rec.Code)
	})

	t.Run("invalid page returns 400", func(t *testing.T) {
		env := newTestEnv(t)
		rec := env.doAs(t, "user-1", http.MethodGet, "/v1/shipping/shipments?page=0", "")
		require.Equal(t, http.StatusBadRequest, rec.Code)
	})

	t.Run("invalid page_size returns 400", func(t *testing.T) {
		env := newTestEnv(t)
		rec := env.doAs(t, "user-1", http.MethodGet, "/v1/shipping/shipments?page_size=101", "")
		require.Equal(t, http.StatusBadRequest, rec.Code)
	})

	t.Run("paginates results", func(t *testing.T) {
		env := newTestEnv(t)
		env.seedShipment(t, "user-1", "order-1", "")
		env.seedShipment(t, "user-1", "order-2", "")
		env.seedShipment(t, "user-1", "order-3", "")

		rec := env.doAs(t, "user-1", http.MethodGet, "/v1/shipping/shipments?page=1&page_size=2", "")
		require.Equal(t, http.StatusOK, rec.Code)

		envl := decodeEnvelope(t, rec)
		var shipments []ShipmentResponse
		require.NoError(t, json.Unmarshal(envl.Data, &shipments))
		assert.Len(t, shipments, 2)
		require.NotNil(t, envl.Meta)
		assert.Equal(t, 3, envl.Meta.Total)
		assert.Equal(t, 2, envl.Meta.TotalPages)
	})
}
