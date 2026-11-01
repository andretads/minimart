package event

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestOrderConfirmation(t *testing.T) {
	h := NewHandler(nil)

	t.Run("malformed JSON returns an error", func(t *testing.T) {
		assert.Error(t, h.OrderConfirmation(t.Context(), []byte("not json")))
	})

	t.Run("valid payload returns nil", func(t *testing.T) {
		msg := []byte(`{"order_id":"o1","user_id":"u1","total_cents":1250,"item_count":2}`)
		assert.NoError(t, h.OrderConfirmation(t.Context(), msg))
	})
}
