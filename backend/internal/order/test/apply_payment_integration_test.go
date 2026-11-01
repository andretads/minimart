//go:build integration

package test

import (
	"encoding/json"
	"minimart/internal/order/application"
	"minimart/internal/order/domain"
	orderevent "minimart/internal/order/presentation/message/event"
	"minimart/pkg/kafka"
	"minimart/pkg/message"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// paymentSucceededPayload builds the Kafka payload the order consumer receives.
func paymentSucceededPayload(t *testing.T, orderID string) []byte {
	t.Helper()
	body, err := json.Marshal(message.PaymentSucceeded{
		PaymentID:   "pay-1",
		OrderID:     orderID,
		AmountCents: 500,
		Currency:    "usd",
		ProviderRef: "fake_ref",
	})
	require.NoError(t, err)
	return body
}

func TestApplyPaymentOutcomeIntegration(t *testing.T) {
	t.Run("Order does not exist", func(t *testing.T) {
		svc, _, _, _ := newOrderService(t)

		err := svc.ApplyPaymentOutcome(t.Context(), uniqueOrderID(), application.PaymentOutcomeSucceeded, "")
		require.ErrorIs(t, err, domain.ErrOrderNotFound)
	})

	t.Run("A duplicate delivery must not change anything", func(t *testing.T) {
		svc, repo, pool, _ := newOrderService(t)
		id := uniqueOrderID()
		seedOrderWithStatus(t, repo, pool, id, "it-user", domain.StatusPaid)
		clearOutbox(t, pool, id)

		h := orderevent.NewHandler(svc)
		payload := paymentSucceededPayload(t, id)
		require.NoError(t, h.PaymentSucceeded(t.Context(), payload))
		require.NoError(t, h.PaymentSucceeded(t.Context(), payload), "redelivery must be a no-op, not an error")

		got, err := repo.GetByID(t.Context(), id)
		require.NoError(t, err)
		assert.Equal(t, domain.StatusPaid, got.Status)
		assert.Zero(t, countOutbox(t.Context(), t, pool, id), "no order write was attempted")
	})

	t.Run("A cancelled order must not be resurrected by a late payment", func(t *testing.T) {
		svc, repo, pool, _ := newOrderService(t)
		id := uniqueOrderID()
		seedOrderWithStatus(t, repo, pool, id, "it-user", domain.StatusCancelled)
		clearOutbox(t, pool, id)

		require.NoError(t, svc.ApplyPaymentOutcome(t.Context(), id, application.PaymentOutcomeSucceeded, ""))

		got, err := repo.GetByID(t.Context(), id)
		require.NoError(t, err)
		assert.Equal(t, domain.StatusCancelled, got.Status)
		assert.Zero(t, countOutbox(t.Context(), t, pool, id), "no order write was attempted")
	})

	t.Run("A failed payment must not be resurrected by a late success", func(t *testing.T) {
		svc, repo, pool, _ := newOrderService(t)
		id := uniqueOrderID()
		seedOrderWithStatus(t, repo, pool, id, "it-user", domain.StatusPaymentFailed)
		clearOutbox(t, pool, id)

		require.NoError(t, svc.ApplyPaymentOutcome(t.Context(), id, application.PaymentOutcomeSucceeded, ""))

		got, err := repo.GetByID(t.Context(), id)
		require.NoError(t, err)
		assert.Equal(t, domain.StatusPaymentFailed, got.Status)
		assert.Zero(t, countOutbox(t.Context(), t, pool, id), "no order write was attempted")
	})

	t.Run("An unknown payment can still be resolved", func(t *testing.T) {
		svc, repo, pool, _ := newOrderService(t)
		id := uniqueOrderID()
		seedOrderWithStatus(t, repo, pool, id, "it-user", domain.StatusPaymentUnknown)
		clearOutbox(t, pool, id)

		require.NoError(t, svc.ApplyPaymentOutcome(t.Context(), id, application.PaymentOutcomeSucceeded, ""))

		got, err := repo.GetByID(t.Context(), id)
		require.NoError(t, err)
		assert.Equal(t, domain.StatusPaid, got.Status)
	})

	t.Run("Payment succeeded marks the order paid", func(t *testing.T) {
		svc, repo, pool, _ := newOrderService(t)
		id := uniqueOrderID()
		seedOrder(t, repo, pool, id, "it-user")
		clearOutbox(t, pool, id)

		require.NoError(t, svc.ApplyPaymentOutcome(t.Context(), id, application.PaymentOutcomeSucceeded, ""))

		got, err := repo.GetByID(t.Context(), id)
		require.NoError(t, err)
		assert.Equal(t, domain.StatusPaid, got.Status)
		assert.Zero(t, countOutboxTopic(t.Context(), t, pool, id, kafka.OrderStockReleaseTopic.String()),
			"paid stock stays reserved")
	})

	t.Run("Payment failed marks the order payment_failed and releases stock", func(t *testing.T) {
		svc, repo, pool, _ := newOrderService(t)
		id := uniqueOrderID()
		seedOrder(t, repo, pool, id, "it-user")
		clearOutbox(t, pool, id)

		require.NoError(t, svc.ApplyPaymentOutcome(t.Context(), id, application.PaymentOutcomeFailed, ""))

		got, err := repo.GetByID(t.Context(), id)
		require.NoError(t, err)
		assert.Equal(t, domain.StatusPaymentFailed, got.Status)
		assert.ElementsMatch(t, []string{kafka.OrderStockReleaseTopic.String()}, outboxTopics(t.Context(), t, pool, id))
	})

	t.Run("Payment unknown marks the order payment_unknown and holds stock", func(t *testing.T) {
		svc, repo, pool, _ := newOrderService(t)
		id := uniqueOrderID()
		seedOrder(t, repo, pool, id, "it-user")
		clearOutbox(t, pool, id)

		require.NoError(t, svc.ApplyPaymentOutcome(t.Context(), id, application.PaymentOutcomeUnknown, ""))

		got, err := repo.GetByID(t.Context(), id)
		require.NoError(t, err)
		assert.Equal(t, domain.StatusPaymentUnknown, got.Status)
		assert.Zero(t, countOutboxTopic(t.Context(), t, pool, id, kafka.OrderStockReleaseTopic.String()))
	})

	t.Run("An order without shipping hands nothing over to the shipping domain", func(t *testing.T) {
		svc, repo, pool, _ := newOrderService(t)
		id := uniqueOrderID()
		seedOrder(t, repo, pool, id, "it-user")
		clearOutbox(t, pool, id)

		require.NoError(t, svc.ApplyPaymentOutcome(t.Context(), id, application.PaymentOutcomeSucceeded, ""))

		got, err := repo.GetByID(t.Context(), id)
		require.NoError(t, err)
		assert.Equal(t, domain.StatusPaid, got.Status)
		assert.Zero(t, countOutboxTopic(t.Context(), t, pool, id, kafka.OrderPaidTopic.String()))
		assert.Zero(t, countOutboxTopic(t.Context(), t, pool, id, kafka.OrderStockReleaseTopic.String()))
	})

	t.Run("Payment succeeded on an order with shipping hands the snapshot over", func(t *testing.T) {
		svc, repo, pool, _ := newOrderService(t)
		id := uniqueOrderID()
		seedShippingOrder(t, repo, pool, id, "it-user")
		clearOutbox(t, pool, id)

		require.NoError(t, svc.ApplyPaymentOutcome(t.Context(), id, application.PaymentOutcomeSucceeded, ""))

		got, err := repo.GetByID(t.Context(), id)
		require.NoError(t, err)
		assert.Equal(t, domain.StatusPaid, got.Status)

		assert.Equal(t, 1, countOutboxTopic(t.Context(), t, pool, id, kafka.OrderPaidTopic.String()))
		var event message.OrderPaid
		require.NoError(t, json.Unmarshal(outboxBody(t.Context(), t, pool, id, kafka.OrderPaidTopic.String()), &event))
		assert.Equal(t, id, event.OrderID)
		assert.Equal(t, "it-user", event.UserID)
		assert.Equal(t, int64(750), event.TotalCents)
		assert.Equal(t, "q1", event.ShippingQuoteID)
		assert.Equal(t, "m1", event.ShippingMethodID)
		assert.Equal(t, featureDestination(), event.DestinationSnapshot)
	})
}
