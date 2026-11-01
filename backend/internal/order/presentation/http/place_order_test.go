package http

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPlaceOrderStatusMapping(t *testing.T) {
	tests := []struct {
		name       string
		body       string
		wantStatus int
	}{
		{name: "corpo JSON inválido", body: `{`, wantStatus: http.StatusBadRequest},
		{name: "no items", body: `{"user_id":"u1","items":[]}`, wantStatus: http.StatusBadRequest},
		{name: "unknown product", body: `{"user_id":"u1","items":[{"product_id":"nope","quantity":1}]}`, wantStatus: http.StatusBadRequest},
		{name: "non-positive quantity", body: `{"user_id":"u1","items":[{"product_id":"p1","quantity":0}]}`, wantStatus: http.StatusBadRequest},
		{name: "valid", body: `{"user_id":"u1","items":[{"product_id":"p1","quantity":2},{"product_id":"p2","quantity":1}]}`, wantStatus: http.StatusCreated},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mux := newTestMux(t)
			rec := do(t, mux, http.MethodPost, "/v1/orders", tt.body)
			assert.Equal(t, tt.wantStatus, rec.Code)

			var env envelope
			require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &env))
			if tt.wantStatus != http.StatusCreated {
				assert.False(t, env.Success)
				require.NotNil(t, env.Error)
				assert.Equal(t, tt.wantStatus, env.Error.Code)
			}
		})
	}
}

func TestPlaceOrderReturnsOrder(t *testing.T) {
	mux := newTestMux(t)

	rec := do(t, mux, http.MethodPost, "/v1/orders",
		`{"user_id":"u1","items":[{"product_id":"p1","quantity":2},{"product_id":"p2","quantity":1}]}`)
	require.Equal(t, http.StatusCreated, rec.Code)

	var env struct {
		Data orderResponse `json:"data"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &env))
	assert.NotEmpty(t, env.Data.ID)
	assert.Equal(t, "u1", env.Data.UserID)
	assert.Equal(t, int64(1250), env.Data.TotalCents)
	require.Len(t, env.Data.Items, 2)
	assert.Equal(t, int64(1000), env.Data.Items[0].LineTotalCents)
	assert.NotEmpty(t, env.Data.CreatedAt)
}

func TestPlaceOrderRequiresIdempotencyKey(t *testing.T) {
	mux := newTestMux(t)

	rec := doWithKey(t, mux, http.MethodPost, "/v1/orders",
		`{"user_id":"u1","items":[{"product_id":"p1","quantity":1}]}`, "")
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestPlaceOrderReplaysWithSameKey(t *testing.T) {
	mux := newTestMux(t)
	body := `{"user_id":"u1","items":[{"product_id":"p1","quantity":1}]}`

	first := doWithKey(t, mux, http.MethodPost, "/v1/orders", body, "k1")
	require.Equal(t, http.StatusCreated, first.Code)
	second := doWithKey(t, mux, http.MethodPost, "/v1/orders", body, "k1")
	require.Equal(t, http.StatusCreated, second.Code)

	var a, b struct {
		Data orderResponse `json:"data"`
	}
	require.NoError(t, json.Unmarshal(first.Body.Bytes(), &a))
	require.NoError(t, json.Unmarshal(second.Body.Bytes(), &b))
	assert.Equal(t, a.Data.ID, b.Data.ID, "a retry with the same key returns the original order")
}
