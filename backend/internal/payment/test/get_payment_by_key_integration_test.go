//go:build integration

package test

import (
	"minimart/internal/payment/domain"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetPaymentByKeyIntegration(t *testing.T) {
	t.Run("A client recovers the same payment by its idempotency key", func(t *testing.T) {
		e := newPaymentEnv(t)
		req := e.request()
		created := e.createPayment(t, req)

		got, err := e.svc.GetPaymentByKey(t.Context(), req.IdempotencyKey, e.userID)
		require.NoError(t, err)
		assert.Equal(t, created.ID, got.ID)
		assert.Equal(t, created.Status, got.Status)
	})

	t.Run("Another user cannot read a payment by idempotency key", func(t *testing.T) {
		e := newPaymentEnv(t)
		req := e.request()
		e.createPayment(t, req)

		_, err := e.svc.GetPaymentByKey(t.Context(), req.IdempotencyKey, e.userID+"-other")
		require.ErrorIs(t, err, domain.ErrPaymentNotFound)
	})

	t.Run("Unknown idempotency key is not found", func(t *testing.T) {
		e := newPaymentEnv(t)

		_, err := e.svc.GetPaymentByKey(t.Context(), "missing-"+uniquePaymentKey(), e.userID)
		require.ErrorIs(t, err, domain.ErrPaymentNotFound)
	})
}
