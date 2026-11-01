package application

import (
	"minimart/internal/payment/domain"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetPayment(t *testing.T) {
	t.Run("Payment does not exist", func(t *testing.T) {
		e := newTestEnv(t)

		_, err := e.svc.GetPayment(t.Context(), "missing")
		require.Error(t, err)
		require.ErrorIs(t, err, domain.ErrPaymentNotFound)
	})

	t.Run("Payment got returned with no problems", func(t *testing.T) {
		e := newTestEnv(t)
		e.useOrder(defaultOrder())

		created, err := e.svc.CreatePayment(t.Context(), validCreatePaymentRequest())
		require.NoError(t, err)

		got, err := e.svc.GetPayment(t.Context(), created.ID)
		require.NoError(t, err)
		assert.Equal(t, created.ID, got.ID)
		assert.Equal(t, "o1", got.OrderID)
		assert.Equal(t, domain.StatusSucceeded, got.Status)
	})
}
