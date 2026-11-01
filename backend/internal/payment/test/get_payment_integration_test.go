//go:build integration

package test

import (
	"minimart/internal/payment/domain"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetPaymentIntegration(t *testing.T) {
	t.Run("Payment does not exist", func(t *testing.T) {
		e := newPaymentEnv(t)

		_, err := e.svc.GetPayment(t.Context(), "missing")
		require.Error(t, err)
		require.ErrorIs(t, err, domain.ErrPaymentNotFound)
	})

	t.Run("Payment got returned with no problems", func(t *testing.T) {
		e := newPaymentEnv(t)

		created := e.createPayment(t, e.request())

		got, err := e.svc.GetPayment(t.Context(), created.ID)
		require.NoError(t, err)
		assert.Equal(t, created.ID, got.ID)
		assert.Equal(t, e.orderID, got.OrderID)
		assert.Equal(t, domain.StatusSucceeded, got.Status)
	})
}
