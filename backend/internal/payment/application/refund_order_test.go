package application

import (
	"errors"
	"minimart/internal/payment/domain"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRefundOrderPayments(t *testing.T) {
	e := newTestEnv(t)
	e.useOrder(cancelledOrder())
	storedPaymentWithStatus(t, e.repo, "pay-1", "o1", domain.StatusSucceeded)
	storedPaymentWithStatus(t, e.repo, "pay-2", "o1", domain.StatusSucceeded)
	storedPaymentWithStatus(t, e.repo, "pay-other-order", "o2", domain.StatusSucceeded)

	require.NoError(t, e.svc.RefundOrderPayments(t.Context(), "o1"))

	for _, id := range []string{"pay-1", "pay-2"} {
		assert.Equal(t, domain.StatusRefunded, storedPayment(t, e.repo, id).Status)
	}
	assert.Equal(t, domain.StatusSucceeded, storedPayment(t, e.repo, "pay-other-order").Status,
		"payments of other orders are untouched")
	assert.Equal(t, 2, e.provider.refunded)
}

func TestRefundOrderPaymentsSkipsAlreadyRefunded(t *testing.T) {
	e := newTestEnv(t)
	e.useOrder(cancelledOrder())
	storedPaymentWithStatus(t, e.repo, "pay-1", "o1", domain.StatusRefunded)

	require.NoError(t, e.svc.RefundOrderPayments(t.Context(), "o1"))
	assert.Equal(t, 0, e.provider.refunded)
}

func TestRefundOrderPaymentsTreatsLostRaceAsNoOp(t *testing.T) {
	e := newTestEnv(t)
	e.useOrder(cancelledOrder())
	e.repo = &failingRepository{Repository: e.repo, refundErr: domain.ErrPaymentNotRefundable}
	e.wire()
	storedPaymentWithStatus(t, e.repo, "pay-1", "o1", domain.StatusSucceeded)

	require.NoError(t, e.svc.RefundOrderPayments(t.Context(), "o1"), "at-least-once delivery must not fail on a raced refund")
}

func TestRefundOrderPaymentsPropagatesProviderFailure(t *testing.T) {
	e := newTestEnv(t)
	e.useOrder(cancelledOrder())
	e.provider.refundErr = errors.New("gateway down")
	e.wire()
	storedPaymentWithStatus(t, e.repo, "pay-1", "o1", domain.StatusSucceeded)

	err := e.svc.RefundOrderPayments(t.Context(), "o1")
	require.Error(t, err)
	require.ErrorIs(t, err, ErrRefundFailed)
}

func TestRefundOrderPaymentsUnknownOrderIsNoOp(t *testing.T) {
	e := newTestEnv(t)

	require.NoError(t, e.svc.RefundOrderPayments(t.Context(), "no-such-order"))
	assert.Equal(t, 0, e.provider.refunded)
}
