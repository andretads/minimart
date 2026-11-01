package event

import (
	"minimart/internal/order/domain"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPaymentUnknown(t *testing.T) {
	h, repo := seededHandler(t)

	require.NoError(t, h.PaymentUnknown(t.Context(), []byte(`{"payment_id":"pay1","order_id":"o1","amount_cents":100,"currency":"usd","reason":"timeout"}`)))
	got, err := repo.GetByID(t.Context(), "o1")
	require.NoError(t, err)
	assert.Equal(t, domain.StatusPaymentUnknown, got.Status)
}

func TestPaymentUnknownMalformedJSON(t *testing.T) {
	h := NewHandler(nil)
	assert.Error(t, h.PaymentUnknown(t.Context(), []byte("not json")))
}
