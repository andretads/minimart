package event

import (
	"minimart/pkg/metrics"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOrderCancelled(t *testing.T) {
	t.Run("malformed JSON returns an error and sends nothing", func(t *testing.T) {
		m := &countingMetrics{}
		h := NewHandler(newTestService(m))

		require.Error(t, h.OrderCancelled(t.Context(), []byte("not json")))
		assert.Empty(t, m.sent)
	})

	t.Run("valid payload triggers a successful cancellation", func(t *testing.T) {
		m := &countingMetrics{}
		h := NewHandler(newTestService(m))

		msg := []byte(`{"order_id":"o1","user_id":"u1","total_cents":1250,"item_count":2}`)
		require.NoError(t, h.OrderCancelled(t.Context(), msg))
		assert.Equal(t, []string{metrics.StatusSuccess}, m.sent)
	})
}
