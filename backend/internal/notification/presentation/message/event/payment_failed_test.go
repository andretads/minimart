package event

import (
	"minimart/pkg/metrics"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPaymentFailed(t *testing.T) {
	t.Run("malformed JSON returns an error and sends nothing", func(t *testing.T) {
		m := &countingMetrics{}
		h := NewHandler(newTestService(m))

		require.Error(t, h.PaymentFailed(t.Context(), []byte("not json")))
		assert.Empty(t, m.sent)
	})

	t.Run("valid payload triggers a successful notification", func(t *testing.T) {
		m := &countingMetrics{}
		h := NewHandler(newTestService(m))

		msg := []byte(`{"payment_id":"p1","order_id":"o1","user_id":"u1","amount_cents":1250,"currency":"USD","reason":"declined"}`)
		require.NoError(t, h.PaymentFailed(t.Context(), msg))
		assert.Equal(t, []string{metrics.StatusSuccess}, m.sent)
	})
}
