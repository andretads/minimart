//go:build integration

package test

import (
	"encoding/json"
	"errors"
	"minimart/internal/payment/application"
	"minimart/internal/payment/domain"
	"minimart/internal/payment/infrastructure/provider/fake"
	"minimart/pkg/kafka"
	"minimart/pkg/message"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRefundPaymentIntegration(t *testing.T) {
	t.Run("Payment does not exist", func(t *testing.T) {
		e := newPaymentEnv(t)

		_, err := e.svc.RefundPayment(t.Context(), "missing")
		require.Error(t, err)
		require.ErrorIs(t, err, domain.ErrPaymentNotFound)
	})

	t.Run("Payment is not succeeded", func(t *testing.T) {
		e := newPaymentEnv(t)
		e.reader.order.TotalCents = 9999

		req := e.request()
		req.AmountCents = 9999
		created := e.createPayment(t, req)
		require.Equal(t, domain.StatusFailed, e.stored(t, created.ID).Status)

		_, err := e.svc.RefundPayment(t.Context(), created.ID)
		require.Error(t, err)
		require.ErrorIs(t, err, domain.ErrPaymentNotRefundable)

		rows := outboxRows(t, e.pool, created.ID)
		assert.Equal(t, 0, countTopic(rows, kafka.PaymentRefundedTopic))
	})

	t.Run("Order is not cancelled", func(t *testing.T) {
		e := newPaymentEnv(t)
		e.setOrderStatus("paid")

		created := e.createPayment(t, e.request())

		_, err := e.svc.RefundPayment(t.Context(), created.ID)
		require.Error(t, err)
		require.ErrorIs(t, err, application.ErrOrderNotCancelled)

		rows := outboxRows(t, e.pool, created.ID)
		assert.Equal(t, 0, countTopic(rows, kafka.PaymentRefundedTopic))
	})

	t.Run("Provider fails the refund", func(t *testing.T) {
		e := newPaymentEnv(t)
		e.provider = failingRefundProvider{Provider: fake.New(fake.Config{}), err: errors.New("gateway down")}
		e.wire()

		created := e.createPayment(t, e.request())
		e.setOrderStatus("cancelled")

		_, err := e.svc.RefundPayment(t.Context(), created.ID)
		require.Error(t, err)
		require.ErrorIs(t, err, application.ErrRefundFailed)

		assert.Equal(t, domain.StatusSucceeded, e.stored(t, created.ID).Status)
	})

	t.Run("Payment is already refunded", func(t *testing.T) {
		e := newPaymentEnv(t)
		created := e.createPayment(t, e.request())
		e.setOrderStatus("cancelled")
		refunded, err := e.svc.RefundPayment(t.Context(), created.ID)
		require.NoError(t, err)
		require.Equal(t, domain.StatusRefunded, refunded.Status)

		again, err := e.svc.RefundPayment(t.Context(), created.ID)
		require.NoError(t, err)
		assert.Equal(t, domain.StatusRefunded, again.Status)
	})

	t.Run("Refund loses a race, so the payment was already refunded", func(t *testing.T) {
		e := newPaymentEnv(t)
		created := e.createPayment(t, e.request())
		e.setOrderStatus("cancelled")
		// Another worker refunds first; replaying the event must be a no-op.
		_, err := e.svc.RefundPayment(t.Context(), created.ID)
		require.NoError(t, err)

		require.NoError(t, e.svc.RefundOrderPayments(t.Context(), created.OrderID))
		assert.Equal(t, domain.StatusRefunded, e.stored(t, created.ID).Status)
	})

	t.Run("Succeeded payment on a cancelled order is refunded", func(t *testing.T) {
		e := newPaymentEnv(t)
		created := e.createPayment(t, e.request())
		e.setOrderStatus("cancelled")

		got, err := e.svc.RefundPayment(t.Context(), created.ID)
		require.NoError(t, err)
		assert.Equal(t, domain.StatusRefunded, got.Status)
		assert.Equal(t, domain.StatusRefunded, e.stored(t, created.ID).Status)

		rows := outboxRows(t, e.pool, created.ID)
		assert.Equal(t, 1, countTopic(rows, kafka.PaymentRefundedTopic))

		var refunded message.PaymentRefunded
		for _, row := range rows {
			if row.Topic == kafka.PaymentRefundedTopic.String() {
				require.NoError(t, json.Unmarshal(row.Body, &refunded))
			}
		}
		assert.Equal(t, "reembolso solicitado", refunded.Reason)
	})
}
