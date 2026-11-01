package carthttp

import (
	"bytes"
	"minimart/internal/cart/application"
	"minimart/pkg/response"
	"net/http"
	"testing"

	catalogclient "minimart/internal/catalog/presentation/module/client"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHandlers_AddItem(t *testing.T) {
	mux := newTestMux(t)

	t.Run("adds item successfully", func(t *testing.T) {
		body := bytes.NewBufferString(`{"product_id":"prod-1","quantity":2}`)
		rec := doRequest(t, mux, http.MethodPost, "/v1/cart/items?user_id=user-1", body.Bytes(), map[string]string{"Content-Type": "application/json"})
		assert.Equal(t, http.StatusOK, rec.Code)
		var r response.Response
		decodeResponse(t, rec, &r)
		assert.True(t, r.Success)
	})

	t.Run("increments existing item", func(t *testing.T) {
		// First add
		body := bytes.NewBufferString(`{"product_id":"prod-1","quantity":2}`)
		rec := doRequest(t, mux, http.MethodPost, "/v1/cart/items?user_id=user-1", body.Bytes(), map[string]string{"Content-Type": "application/json"})
		require.Equal(t, http.StatusOK, rec.Code)

		// Second add (increment)
		body = bytes.NewBufferString(`{"product_id":"prod-1","quantity":3}`)
		rec = doRequest(t, mux, http.MethodPost, "/v1/cart/items?user_id=user-1", body.Bytes(), map[string]string{"Content-Type": "application/json"})
		assert.Equal(t, http.StatusOK, rec.Code)
	})

	t.Run("invalid quantity returns 400", func(t *testing.T) {
		body := bytes.NewBufferString(`{"product_id":"prod-1","quantity":0}`)
		rec := doRequest(t, mux, http.MethodPost, "/v1/cart/items?user_id=user-1", body.Bytes(), map[string]string{"Content-Type": "application/json"})
		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})

	t.Run("product not found returns 400", func(t *testing.T) {
		body := bytes.NewBufferString(`{"product_id":"prod-999","quantity":1}`)
		rec := doRequest(t, mux, http.MethodPost, "/v1/cart/items?user_id=user-1", body.Bytes(), map[string]string{"Content-Type": "application/json"})
		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})

	t.Run("unauthenticated returns 401", func(t *testing.T) {
		body := bytes.NewBufferString(`{"product_id":"prod-1","quantity":1}`)
		rec := doRequestUnauth(t, mux, http.MethodPost, "/v1/cart/items", body.Bytes(), map[string]string{"Content-Type": "application/json"})
		assert.Equal(t, http.StatusUnauthorized, rec.Code)
	})
}

func TestService_AddItem_Direct(t *testing.T) {
	svc := newTestService(t)
	_, err := svc.AddItem(t.Context(), application.AddItemRequest{UserID: "user-1", ProductID: "prod-999", Quantity: 1})
	require.Error(t, err)
	assert.ErrorIs(t, err, catalogclient.ErrProductNotFound)
}
