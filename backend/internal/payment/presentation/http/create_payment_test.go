package http

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCreatePaymentStatusMapping(t *testing.T) {
	tests := []struct {
		name       string
		body       string
		wantStatus int
	}{
		{name: "corpo JSON inválido", body: `{`, wantStatus: http.StatusBadRequest},
		{name: "negative amount", body: `{"order_id":"o1","user_id":"u1","amount_cents":-1,"currency":"USD","method":"card"}`, wantStatus: http.StatusBadRequest},
		{name: "zero amount", body: `{"order_id":"o1","user_id":"u1","amount_cents":0,"currency":"USD","method":"card"}`, wantStatus: http.StatusBadRequest},
		{name: "empty user", body: `{"order_id":"o1", "user_id":"", "amount_cents":100, "currency":"USD", "method":"card"}`, wantStatus: http.StatusBadRequest},
		{name: "empty currency", body: `{"order_id":"o1","user_id":"u1","amount_cents":100,"currency":"","method":"card"}`, wantStatus: http.StatusBadRequest},
		{name: "empty method", body: `{"order_id":"o1","user_id":"u1","amount_cents":100,"currency":"USD","method":""}`, wantStatus: http.StatusBadRequest},
		{name: "valid", body: `{"order_id":"o1","user_id":"u1","amount_cents":1500,"currency":"USD","method":"card"}`, wantStatus: http.StatusCreated},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mux := newDefaultMux(t)
			rec := do(t, mux, http.MethodPost, "/v1/payments", "k-"+tt.name, tt.body)
			assert.Equal(t, tt.wantStatus, rec.Code)

			var env envelope
			require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &env))
			if tt.wantStatus == http.StatusCreated {
				assert.True(t, env.Success)
			} else {
				assert.False(t, env.Success)
				require.NotNil(t, env.Error)
				assert.Equal(t, tt.wantStatus, env.Error.Code)
			}
		})
	}
}

func TestCreatePaymentOutcomes(t *testing.T) {
	tests := []struct {
		name       string
		amount     int64
		wantStatus string
	}{
		{name: "approved becomes succeeded", amount: 100, wantStatus: "succeeded"},
		{name: "declined becomes failed", amount: 400, wantStatus: "failed"},
		{name: "timeout becomes unknown", amount: 500, wantStatus: "unknown"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mux := newDefaultMux(t)
			body := `{"order_id":"o1","user_id":"u1","amount_cents":` + itoa(tt.amount) + `,"currency":"USD","method":"card"}`
			rec := do(t, mux, http.MethodPost, "/v1/payments", "k-"+tt.name, body)
			require.Equal(t, http.StatusCreated, rec.Code)

			var env struct {
				Data paymentResponse `json:"data"`
			}
			require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &env))
			assert.Equal(t, tt.wantStatus, env.Data.Status)
		})
	}
}

func TestCreatePaymentReplayAndConflict(t *testing.T) {
	mux := newDefaultMux(t)
	body := `{"order_id":"o1","user_id":"u1","amount_cents":1500,"currency":"USD","method":"card"}`

	first := do(t, mux, http.MethodPost, "/v1/payments", "dup", body)
	require.Equal(t, http.StatusCreated, first.Code)
	var created struct {
		Data paymentResponse `json:"data"`
	}
	require.NoError(t, json.Unmarshal(first.Body.Bytes(), &created))

	t.Run("replay returns the same payment", func(t *testing.T) {
		again := do(t, mux, http.MethodPost, "/v1/payments", "dup", body)
		require.Equal(t, http.StatusCreated, again.Code)
		var env struct {
			Data paymentResponse `json:"data"`
		}
		require.NoError(t, json.Unmarshal(again.Body.Bytes(), &env))
		assert.Equal(t, created.Data.ID, env.Data.ID)
	})

	t.Run("different params conflict", func(t *testing.T) {
		conflict := do(t, mux, http.MethodPost, "/v1/payments", "dup",
			`{"order_id":"o1","user_id":"u1","amount_cents":2500,"currency":"USD","method":"card"}`)
		assert.Equal(t, http.StatusConflict, conflict.Code)
	})
}
