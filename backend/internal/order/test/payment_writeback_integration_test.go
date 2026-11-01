//go:build integration

package test

import (
	"context"
	"minimart/internal/order/application"
	"minimart/internal/order/domain"
	orderevent "minimart/internal/order/presentation/message/event"
	"minimart/pkg/kafka"
	"minimart/pkg/message"
	"minimart/pkg/metrics"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestPaymentWriteBack_ConsumerUpdatesOrderStatus proves the end-to-end path from
// a payment-succeeded message on Kafka through the order consumer to the order's
// status in real Postgres.
func TestPaymentWriteBack_ConsumerUpdatesOrderStatus(t *testing.T) {
	repo, pool := newOrderRepo(t)
	id := uniqueOrderID()
	cleanup(t, pool, id)

	order, err := domain.NewOrder(id, "it-user", "", "", []domain.Item{
		{ProductID: "p1", Name: "Widget", UnitPriceCents: 500, Quantity: 1},
	})
	require.NoError(t, err)
	require.NoError(t, repo.Save(t.Context(), order))

	handler := orderevent.NewHandler(application.NewService(repo, nil, nil, metrics.Noop{}, 0))

	// A per-run topic and group keep this test isolated from any other consumer
	// sharing the payment-succeeded topic.
	topic := kafka.Topic("payment-succeeded-writeback-" + id)
	group := "order-payment-it-" + id

	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()

	consumer := kafka.NewConsumer(testKafkaBrokers, nil, metrics.Noop{})
	errCh := make(chan error, 1)
	go func() {
		errCh <- consumer.Subscribe(ctx, topic, group, handler.PaymentSucceeded)
	}()

	producer, err := kafka.NewProducer(testKafkaBrokers, metrics.Noop{})
	require.NoError(t, err)
	t.Cleanup(func() { _ = producer.Close() })

	require.NoError(t, producer.PublishMessage(t.Context(), topic, id, message.PaymentSucceeded{
		PaymentID:   "pay-1",
		OrderID:     id,
		AmountCents: 500,
		Currency:    "usd",
		ProviderRef: "fake_ref",
	}))

	require.Eventually(t, func() bool {
		got, err := repo.GetByID(t.Context(), id)
		return err == nil && got.Status == domain.StatusPaid
	}, 10*time.Second, 100*time.Millisecond, "order should be paid after the payment-succeeded event is consumed")

	cancel()
	select {
	case err := <-errCh:
		require.NoError(t, err)
	case <-time.After(5 * time.Second):
		t.Fatal("consumer did not stop after cancel")
	}
}

// TestPaymentWriteBack_RedeliveryIsIdempotent processes the same payment-succeeded
// event twice and proves the second delivery writes nothing and does not error.
func TestPaymentWriteBack_RedeliveryIsIdempotent(t *testing.T) {
	repo, pool := newOrderRepo(t)
	id := uniqueOrderID()
	cleanup(t, pool, id)

	order, err := domain.NewOrder(id, "it-user", "", "", []domain.Item{
		{ProductID: "p1", Name: "Widget", UnitPriceCents: 500, Quantity: 1},
	})
	require.NoError(t, err)
	require.NoError(t, repo.Save(t.Context(), order))

	h := orderevent.NewHandler(application.NewService(repo, nil, nil, metrics.Noop{}, 0))
	payload := []byte(`{"payment_id":"pay-1","order_id":"` + id + `","amount_cents":500,"currency":"usd","provider_ref":"fake_ref"}`)

	require.NoError(t, h.PaymentSucceeded(t.Context(), payload))
	require.NoError(t, h.PaymentSucceeded(t.Context(), payload), "redelivery must be a no-op, not an error")

	got, err := repo.GetByID(t.Context(), id)
	require.NoError(t, err)
	assert.Equal(t, domain.StatusPaid, got.Status)
}
