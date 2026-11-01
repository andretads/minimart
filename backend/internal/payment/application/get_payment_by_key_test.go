package application

import (
	"minimart/internal/payment/domain"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetPaymentByKey(t *testing.T) {
	t.Run("A client recovers the same payment by its idempotency key", func(t *testing.T) {
		e := newTestEnv(t)
		e.useOrder(defaultOrder())

		created, err := e.svc.CreatePayment(t.Context(), validCreatePaymentRequest())
		require.NoError(t, err)

		got, err := e.svc.GetPaymentByKey(t.Context(), "k1", "u1")
		require.NoError(t, err)
		assert.Equal(t, created.ID, got.ID)
		assert.Equal(t, created.Status, got.Status)
	})

	t.Run("Another user cannot read a payment by idempotency key", func(t *testing.T) {
		e := newTestEnv(t)
		e.useOrder(defaultOrder())

		_, err := e.svc.CreatePayment(t.Context(), validCreatePaymentRequest())
		require.NoError(t, err)

		_, err = e.svc.GetPaymentByKey(t.Context(), "k1", "u2")
		require.ErrorIs(t, err, domain.ErrPaymentNotFound)
	})

	t.Run("Unknown idempotency key is not found", func(t *testing.T) {
		e := newTestEnv(t)

		_, err := e.svc.GetPaymentByKey(t.Context(), "missing", "u1")
		require.ErrorIs(t, err, domain.ErrPaymentNotFound)
	})
}
