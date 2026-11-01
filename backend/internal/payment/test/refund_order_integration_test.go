//go:build integration

package test

import (
	"minimart/internal/payment/application"
	"minimart/internal/payment/domain"
	"minimart/pkg/metrics"
	"testing"
	"time"

	orderclient "minimart/internal/order/presentation/module/client"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRefundOrderPaymentsIntegration(t *testing.T) {
	t.Run("Refunds every captured payment of a cancelled order", func(t *testing.T) {
		e := newPaymentEnv(t)

		orderID := "it-order-" + uniquePaymentKey()
		e.reader.order.ID = orderID
		e.reader.order.TotalCents = 1500
		e.wire()

		var ids []string
		for range 2 {
			req := e.request()
			req.OrderID = orderID
			created := e.createPayment(t, req)
			require.Equal(t, domain.StatusSucceeded, created.Status)
			ids = append(ids, created.ID)
		}

		e.setOrderStatus("cancelled")
		require.NoError(t, e.svc.RefundOrderPayments(t.Context(), orderID))
		for _, id := range ids {
			assert.Equal(t, domain.StatusRefunded, e.stored(t, id).Status)
		}

		e.setOrderStatus("cancelled")
		require.NoError(t, e.svc.RefundOrderPayments(t.Context(), orderID))
	})

	t.Run("Leaves other orders untouched", func(t *testing.T) {
		e := newPaymentEnv(t)

		orderID := "it-order-" + uniquePaymentKey()
		e.reader.order.ID = orderID
		e.reader.order.TotalCents = 1500
		e.wire()

		req := e.request()
		req.OrderID = orderID
		created := e.createPayment(t, req)
		require.Equal(t, domain.StatusSucceeded, created.Status)

		otherOrderID := "it-order-" + uniquePaymentKey()
		otherReader := &mutableOrderReader{order: orderclient.Order{ID: otherOrderID, UserID: e.userID, Status: "placed", TotalCents: 1500, Currency: "USD"}}
		// keep e.reader for first order; create other via temp service
		otherSvc := application.NewService(e.repo, otherReader, e.idem, e.provider, metrics.Noop{}, time.Hour)
		otherReq := application.CreatePaymentRequest{OrderID: otherOrderID, UserID: e.userID, IdempotencyKey: uniquePaymentKey(), AmountCents: 1500, Currency: "USD", Method: "card"}
		otherCreated, err := otherSvc.CreatePayment(t.Context(), otherReq)
		require.NoError(t, err)
		cleanup(t, e.pool, otherCreated.ID, otherReq.IdempotencyKey)

		e.setOrderStatus("cancelled")
		require.NoError(t, e.svc.RefundOrderPayments(t.Context(), orderID))
		assert.Equal(t, domain.StatusRefunded, e.stored(t, created.ID).Status)
		assert.Equal(t, domain.StatusSucceeded, e.stored(t, otherCreated.ID).Status)
	})
}
