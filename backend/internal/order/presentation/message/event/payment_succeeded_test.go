package event

import (
	"minimart/internal/order/domain"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPaymentSucceeded(t *testing.T) {
	h, repo := seededHandler(t)

	require.NoError(t, h.PaymentSucceeded(t.Context(), []byte(`{"payment_id":"pay1","order_id":"o1","amount_cents":100,"currency":"usd","provider_ref":"ref"}`)))
	got, err := repo.GetByID(t.Context(), "o1")
	require.NoError(t, err)
	assert.Equal(t, domain.StatusPaid, got.Status)
}

func TestPaymentSucceededMalformedJSON(t *testing.T) {
	h := NewHandler(nil)
	assert.Error(t, h.PaymentSucceeded(t.Context(), []byte("not json")))
}
