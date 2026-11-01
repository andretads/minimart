package application

import (
	"errors"
	"minimart/internal/payment/domain"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRefundPayment(t *testing.T) {
	t.Run("Payment does not exist", func(t *testing.T) {
		e := newTestEnv(t)

		_, err := e.svc.RefundPayment(t.Context(), "missing")
		require.Error(t, err)
		require.ErrorIs(t, err, domain.ErrPaymentNotFound)
	})

	t.Run("Payment is not succeeded", func(t *testing.T) {
		e := newTestEnv(t)
		e.useOrder(defaultOrder())
		p := storedPaymentWithStatus(t, e.repo, "pay-1", "o1", domain.StatusFailed)

		_, err := e.svc.RefundPayment(t.Context(), p.ID)
		require.Error(t, err)
		require.ErrorIs(t, err, domain.ErrPaymentNotRefundable)
		assert.Equal(t, 0, e.provider.refunded, "the provider is not asked to refund")
	})

	t.Run("Order is not cancelled", func(t *testing.T) {
		e := newTestEnv(t)
		e.useOrder(defaultOrder())
		e.reader.order.Status = "paid"
		e.wire()
		p := storedPaymentWithStatus(t, e.repo, "pay-1", "o1", domain.StatusSucceeded)

		_, err := e.svc.RefundPayment(t.Context(), p.ID)
		require.Error(t, err)
		require.ErrorIs(t, err, ErrOrderNotCancelled)
		assert.Equal(t, 0, e.provider.refunded, "the provider is not asked to refund")
	})

	t.Run("Provider fails the refund", func(t *testing.T) {
		e := newTestEnv(t)
		e.useOrder(cancelledOrder())
		e.provider.refundErr = errors.New("gateway down")
		e.wire()
		p := storedPaymentWithStatus(t, e.repo, "pay-1", "o1", domain.StatusSucceeded)

		_, err := e.svc.RefundPayment(t.Context(), p.ID)
		require.Error(t, err)
		require.ErrorIs(t, err, ErrRefundFailed)

		stored := storedPayment(t, e.repo, p.ID)
		assert.Equal(t, domain.StatusSucceeded, stored.Status, "a failed refund leaves the payment untouched")
	})

	t.Run("Payment is already refunded", func(t *testing.T) {
		e := newTestEnv(t)
		e.useOrder(cancelledOrder())
		p := storedPaymentWithStatus(t, e.repo, "pay-1", "o1", domain.StatusRefunded)

		got, err := e.svc.RefundPayment(t.Context(), p.ID)
		require.NoError(t, err)

		assert.Equal(t, domain.StatusRefunded, got.Status)
		assert.Equal(t, 0, e.provider.refunded, "an already-refunded payment is returned without touching the provider")
	})

	t.Run("Refund loses a race, so the payment was already refunded", func(t *testing.T) {
		e := newTestEnv(t)
		e.useOrder(cancelledOrder())
		succeeded := storedPaymentWithStatus(t, e.repo, "pay-1", "o1", domain.StatusSucceeded)

		// Another worker refunds first; our stale listing still shows succeeded.
		_, err := e.svc.RefundPayment(t.Context(), succeeded.ID)
		require.NoError(t, err)
		e.repo = &losingRefundRepository{Repository: e.repo, listed: []domain.Payment{succeeded}}
		e.wire()

		require.NoError(t, e.svc.RefundOrderPayments(t.Context(), "o1"), "at-least-once delivery must not fail on a raced refund")

		stored := storedPayment(t, e.repo, succeeded.ID)
		assert.Equal(t, domain.StatusRefunded, stored.Status)
	})

	t.Run("Succeeded payment on a cancelled order is refunded", func(t *testing.T) {
		e := newTestEnv(t)
		e.useOrder(cancelledOrder())
		p := storedPaymentWithStatus(t, e.repo, "pay-1", "o1", domain.StatusSucceeded)

		got, err := e.svc.RefundPayment(t.Context(), p.ID)
		require.NoError(t, err)

		assert.Equal(t, domain.StatusRefunded, got.Status)
		stored := storedPayment(t, e.repo, p.ID)
		assert.Equal(t, domain.StatusRefunded, stored.Status)
		require.Equal(t, 1, countEvent[domain.PaymentRefundedEvent](stored.Events()))
		for _, ev := range stored.Events() {
			if refunded, ok := ev.(domain.PaymentRefundedEvent); ok {
				assert.Equal(t, "reembolso solicitado", refunded.Reason)
			}
		}
		assert.Equal(t, 1, e.provider.refunded)
	})
}
