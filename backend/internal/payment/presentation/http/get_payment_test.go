package http

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetPaymentStatusMapping(t *testing.T) {
	mux := newDefaultMux(t)

	rec := do(t, mux, http.MethodPost, "/v1/payments", "k1",
		`{"order_id":"o1","user_id":"u1","amount_cents":1500,"currency":"USD","method":"card"}`)
	require.Equal(t, http.StatusCreated, rec.Code)
	var created struct {
		Data paymentResponse `json:"data"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &created))
	require.NotEmpty(t, created.Data.ID)

	t.Run("found returns the payment", func(t *testing.T) {
		got := do(t, mux, http.MethodGet, "/v1/payments/"+created.Data.ID, "", "")
		require.Equal(t, http.StatusOK, got.Code)

		var env struct {
			Data paymentResponse `json:"data"`
		}
		require.NoError(t, json.Unmarshal(got.Body.Bytes(), &env))
		assert.Equal(t, created.Data.ID, env.Data.ID)
		assert.Equal(t, int64(1500), env.Data.AmountCents)
	})

	t.Run("missing returns 404", func(t *testing.T) {
		got := do(t, mux, http.MethodGet, "/v1/payments/missing", "", "")
		assert.Equal(t, http.StatusNotFound, got.Code)
	})
}
