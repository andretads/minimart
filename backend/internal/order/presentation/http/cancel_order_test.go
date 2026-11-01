package http

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCancelOrderStatusMapping(t *testing.T) {
	mux := newTestMux(t)

	rec := do(t, mux, http.MethodPost, "/v1/orders",
		`{"user_id":"u1","items":[{"product_id":"p1","quantity":1}]}`)
	require.Equal(t, http.StatusCreated, rec.Code)
	var created struct {
		Data orderResponse `json:"data"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &created))
	require.NotEmpty(t, created.Data.ID)

	t.Run("cancels a placed order", func(t *testing.T) {
		got := do(t, mux, http.MethodPost, "/v1/orders/"+created.Data.ID+"/cancel", "")
		require.Equal(t, http.StatusOK, got.Code)

		var env struct {
			Data orderResponse `json:"data"`
		}
		require.NoError(t, json.Unmarshal(got.Body.Bytes(), &env))
		assert.Equal(t, "cancelled", env.Data.Status)
		require.NotNil(t, env.Data.CancelledAt)
	})

	t.Run("cancelling again returns 409", func(t *testing.T) {
		got := do(t, mux, http.MethodPost, "/v1/orders/"+created.Data.ID+"/cancel", "")
		assert.Equal(t, http.StatusConflict, got.Code)
	})

	t.Run("missing returns 404", func(t *testing.T) {
		got := do(t, mux, http.MethodPost, "/v1/orders/missing/cancel", "")
		assert.Equal(t, http.StatusNotFound, got.Code)
	})
}
