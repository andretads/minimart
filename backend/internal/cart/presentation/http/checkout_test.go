package carthttp

import (
	"bytes"
	"minimart/pkg/response"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHandlers_Checkout(t *testing.T) {
	mux := newTestMux(t)

	t.Run("successful checkout", func(t *testing.T) {
		// First add
		body := bytes.NewBufferString(`{"product_id":"prod-1","quantity":2}`)
		rec := doRequest(t, mux, http.MethodPost, "/v1/cart/items?user_id=user-1", body.Bytes(), map[string]string{"Content-Type": "application/json"})
		require.Equal(t, http.StatusOK, rec.Code)

		// Checkout
		checkoutBody := bytes.NewBufferString(`{"shipping_quote_id":"quote-user-1","address_id":"addr-1"}`)
		rec = doRequest(t, mux, http.MethodPost, "/v1/cart/checkout?user_id=user-1", checkoutBody.Bytes(), map[string]string{"Idempotency-Key": "idem-1"})
		assert.Equal(t, http.StatusCreated, rec.Code)
		var r response.Response
		decodeResponse(t, rec, &r)
		assert.True(t, r.Success)
	})

	t.Run("empty cart returns 400", func(t *testing.T) {
		emptyMux := newTestMux(t)

		// Add then clear
		body := bytes.NewBufferString(`{"product_id":"prod-1","quantity":1}`)
		rec := doRequest(t, emptyMux, http.MethodPost, "/v1/cart/items?user_id=user-1", body.Bytes(), map[string]string{"Content-Type": "application/json"})
		require.Equal(t, http.StatusOK, rec.Code)

		rec = doRequest(t, emptyMux, http.MethodDelete, "/v1/cart?user_id=user-1", nil, nil)
		require.Equal(t, http.StatusNoContent, rec.Code)

		// Checkout empty cart
		emptyBody := bytes.NewBufferString(`{"shipping_quote_id":"quote-user-1","address_id":"addr-1"}`)
		rec = doRequest(t, emptyMux, http.MethodPost, "/v1/cart/checkout?user_id=user-1", emptyBody.Bytes(), map[string]string{"Idempotency-Key": "idem-2"})
		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})

	t.Run("missing idempotency key returns 400", func(t *testing.T) {
		// First add
		body := bytes.NewBufferString(`{"product_id":"prod-1","quantity":2}`)
		rec := doRequest(t, mux, http.MethodPost, "/v1/cart/items?user_id=user-1", body.Bytes(), map[string]string{"Content-Type": "application/json"})
		require.Equal(t, http.StatusOK, rec.Code)

		// Checkout without idempotency key
		rec = doRequest(t, mux, http.MethodPost, "/v1/cart/checkout?user_id=user-1", nil, nil)
		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})
}
