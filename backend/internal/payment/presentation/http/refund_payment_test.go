package http

import (
	"context"
	"encoding/json"
	"minimart/internal/order/presentation/module/client"
	"minimart/internal/payment/application"
	"minimart/internal/payment/infrastructure/memory"
	"minimart/internal/payment/infrastructure/provider/fake"
	"minimart/pkg/metrics"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// scriptedOrderReader answers a mutable order status, so a test can cancel the
// order between creating the payment and refunding it.
type scriptedOrderReader struct {
	status     string
	totalCents int64
}

func (r *scriptedOrderReader) GetOrder(_ context.Context, _ string) (client.Order, error) {
	return client.Order{ID: "o1", UserID: "u1", Status: r.status, TotalCents: r.totalCents, Currency: "USD"}, nil
}

func newRefundMux(t *testing.T, reader *scriptedOrderReader) *http.ServeMux {
	t.Helper()
	provider := fake.New(fake.Config{DeclineAmountCents: 400, TimeoutAmountCents: 500})
	svc := application.NewService(memory.New(), reader, newStubIdempotency(), provider, metrics.Noop{}, 24*time.Hour)
	mux := http.NewServeMux()
	NewHandlers(svc).Register(mux)
	return mux
}

func createPayment(t *testing.T, mux *http.ServeMux, key, body string) paymentResponse {
	t.Helper()
	rec := do(t, mux, http.MethodPost, "/v1/payments", key, body)
	require.Equal(t, http.StatusCreated, rec.Code)
	var env struct {
		Data paymentResponse `json:"data"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &env))
	return env.Data
}

func TestRefundPaymentStatusMapping(t *testing.T) {
	reader := &scriptedOrderReader{status: "placed", totalCents: 1500}
	mux := newRefundMux(t, reader)
	created := createPayment(t, mux, "refund-1",
		`{"order_id":"o1","user_id":"u1","amount_cents":1500,"currency":"USD","method":"card"}`)

	reader.status = "cancelled"
	rec := do(t, mux, http.MethodPost, "/v1/payments/"+created.ID+"/refund", "", "")
	require.Equal(t, http.StatusOK, rec.Code)

	var env struct {
		Data paymentResponse `json:"data"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &env))
	assert.Equal(t, created.ID, env.Data.ID)
	assert.Equal(t, "refunded", env.Data.Status)

	t.Run("refund is idempotent", func(t *testing.T) {
		again := do(t, mux, http.MethodPost, "/v1/payments/"+created.ID+"/refund", "", "")
		require.Equal(t, http.StatusOK, again.Code)
		var env struct {
			Data paymentResponse `json:"data"`
		}
		require.NoError(t, json.Unmarshal(again.Body.Bytes(), &env))
		assert.Equal(t, "refunded", env.Data.Status)
	})

	t.Run("missing payment returns 404", func(t *testing.T) {
		missing := do(t, mux, http.MethodPost, "/v1/payments/missing/refund", "", "")
		assert.Equal(t, http.StatusNotFound, missing.Code)
	})
}

func TestRefundPaymentRequiresCancelledOrder(t *testing.T) {
	reader := &scriptedOrderReader{status: "paid", totalCents: 1500}
	mux := newRefundMux(t, reader)
	created := createPayment(t, mux, "refund-paid",
		`{"order_id":"o1","user_id":"u1","amount_cents":1500,"currency":"USD","method":"card"}`)

	rec := do(t, mux, http.MethodPost, "/v1/payments/"+created.ID+"/refund", "", "")
	assert.Equal(t, http.StatusConflict, rec.Code)
}

func TestRefundPaymentRejectsNonSucceeded(t *testing.T) {
	tests := []struct {
		name   string
		amount int64
	}{
		{"failed payment", 400},
		{"unknown payment", 500},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reader := &scriptedOrderReader{status: "placed", totalCents: tt.amount}
			mux := newRefundMux(t, reader)
			body := `{"order_id":"o1","user_id":"u1","amount_cents":` + itoa(tt.amount) + `,"currency":"USD","method":"card"}`
			created := createPayment(t, mux, "refund-"+tt.name, body)

			reader.status = "cancelled"
			rec := do(t, mux, http.MethodPost, "/v1/payments/"+created.ID+"/refund", "", "")
			assert.Equal(t, http.StatusConflict, rec.Code)
		})
	}
}
