package event

import (
	"minimart/pkg/metrics"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestShipmentShipped(t *testing.T) {
	t.Run("malformed JSON returns an error and sends nothing", func(t *testing.T) {
		m := &countingMetrics{}
		h := NewHandler(newTestService(m))

		require.Error(t, h.ShipmentShipped(t.Context(), []byte("not json")))
		assert.Empty(t, m.sent)
	})

	t.Run("valid payload triggers a successful notification", func(t *testing.T) {
		m := &countingMetrics{}
		h := NewHandler(newTestService(m))

		msg := []byte(`{"shipment_id":"s1","order_id":"o1","user_id":"u1","tracking_code":"TRACK123","carrier":"Correios","shipped_at":"2024-01-01T00:00:00Z"}`)
		require.NoError(t, h.ShipmentShipped(t.Context(), msg))
		assert.Equal(t, []string{metrics.StatusSuccess}, m.sent)
	})
}
