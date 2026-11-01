package carthttp

import (
	"bytes"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHandlers_UpdateItem(t *testing.T) {
	mux := newTestMux(t)

	t.Run("updates quantity", func(t *testing.T) {
		// First add
		body := bytes.NewBufferString(`{"product_id":"prod-1","quantity":2}`)
		rec := doRequest(t, mux, http.MethodPost, "/v1/cart/items?user_id=user-1", body.Bytes(), map[string]string{"Content-Type": "application/json"})
		require.Equal(t, http.StatusOK, rec.Code)

		// Update quantity
		body = bytes.NewBufferString(`{"quantity":5}`)
		rec = doRequest(t, mux, http.MethodPatch, "/v1/cart/items/prod-1?user_id=user-1", body.Bytes(), map[string]string{"Content-Type": "application/json"})
		assert.Equal(t, http.StatusOK, rec.Code)
	})

	t.Run("item not found returns 404", func(t *testing.T) {
		body := bytes.NewBufferString(`{"quantity":5}`)
		rec := doRequest(t, mux, http.MethodPatch, "/v1/cart/items/prod-999?user_id=user-1", body.Bytes(), map[string]string{"Content-Type": "application/json"})
		assert.Equal(t, http.StatusNotFound, rec.Code)
	})
}
