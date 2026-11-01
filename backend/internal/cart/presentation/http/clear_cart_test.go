package carthttp

import (
	"bytes"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHandlers_ClearCart(t *testing.T) {
	mux := newTestMux(t)

	t.Run("clears cart", func(t *testing.T) {
		// First add
		body := bytes.NewBufferString(`{"product_id":"prod-1","quantity":2}`)
		rec := doRequest(t, mux, http.MethodPost, "/v1/cart/items?user_id=user-1", body.Bytes(), map[string]string{"Content-Type": "application/json"})
		require.Equal(t, http.StatusOK, rec.Code)

		// Clear cart
		rec = doRequest(t, mux, http.MethodDelete, "/v1/cart?user_id=user-1", nil, nil)
		assert.Equal(t, http.StatusNoContent, rec.Code)
	})
}
