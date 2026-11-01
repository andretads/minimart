package event

import (
	"minimart/internal/order/domain"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPaymentFailed(t *testing.T) {
	h, repo := seededHandler(t)

	require.NoError(t, h.PaymentFailed(t.Context(), []byte(`{"payment_id":"pay1","order_id":"o1","amount_cents":100,"currency":"usd","reason":"declined"}`)))
	got, err := repo.GetByID(t.Context(), "o1")
	require.NoError(t, err)
	assert.Equal(t, domain.StatusPaymentFailed, got.Status)
}

func TestPaymentFailedMalformedJSON(t *testing.T) {
	h := NewHandler(nil)
	assert.Error(t, h.PaymentFailed(t.Context(), []byte("not json")))
}
