//go:build integration

package test

import (
	"context"
	"minimart/internal/notification/application"
	"minimart/internal/notification/domain"
	"minimart/internal/notification/presentation/message/event"
	"minimart/pkg/kafka"
	"minimart/pkg/message"
	"minimart/pkg/metrics"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeMetrics struct {
	metrics.Noop
	mu   sync.Mutex
	sent []string
}

func (f *fakeMetrics) IncNotificationSent(status string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.sent = append(f.sent, status)
}

func (f *fakeMetrics) GetSent() []string {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]string(nil), f.sent...)
}

func TestNotificationConsumer_ConsumesOrderPlaced(t *testing.T) {
	m := &fakeMetrics{}
	repo := newTestRepo(t)
	sender := newFakeEmailSender()
	resolver := newFakeRecipientResolver()
	resolver.SetEmail("test-user-1", "user1@example.com")
	svc := application.NewService(repo, newTestDeliveryRepo(t), newTestPrefsRepo(t), sender, nil, nil, resolver, nil, m, application.RetryConfig{})
	h := event.NewHandler(svc)

	producer, err := kafka.NewProducer(testKafkaBrokers, metrics.Noop{})
	require.NoError(t, err)
	t.Cleanup(func() { _ = producer.Close() })

	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()

	consumer := kafka.NewConsumer(testKafkaBrokers, nil, metrics.Noop{})
	errCh := make(chan error, 1)
	topic := kafka.Topic("order-placed-notification-consumer")
	consumerGroup := "notification-consumer-order-placed"
	go func() {
		errCh <- consumer.Subscribe(ctx, topic, consumerGroup, func(_ context.Context, msg []byte) error {
			if herr := h.OrderPlaced(ctx, msg); herr != nil {
				t.Logf("handler error: %v", herr)
				return herr
			}
			cancel()
			return nil
		})
	}()

	orderPlaced := message.OrderPlaced{
		OrderID:    "test-order-1",
		UserID:     "test-user-1",
		TotalCents: 1500,
		ItemCount:  2,
	}
	require.NoError(t, producer.PublishMessage(t.Context(), topic, orderPlaced.OrderID, orderPlaced))

	select {
	case <-ctx.Done():
	case subErr := <-errCh:
		require.NoError(t, subErr)
	case <-time.After(10 * time.Second):
		t.Fatal("timeout waiting for message")
	}

	assert.Equal(t, []string{metrics.StatusSuccess}, m.GetSent())

	stored, err := repo.ListByUserID(t.Context(), "test-user-1", 1, 20)
	require.NoError(t, err)
	require.Len(t, stored, 1, "the consumed event must be recorded exactly once")
	assert.Equal(t, "test-order-1", stored[0].OrderID)
	assert.Equal(t, domain.TypeOrderConfirmation, stored[0].Type)
	assert.Contains(t, stored[0].Body, "R$ 15,00", "the event payload survived the round trip")
}

func TestNotificationConsumer_ConsumesPaymentSucceeded(t *testing.T) {
	m := &fakeMetrics{}
	repo := newTestRepo(t)
	sender := newFakeEmailSender()
	resolver := newFakeRecipientResolver()
	resolver.SetEmail("test-user-2", "user2@example.com")
	svc := application.NewService(repo, newTestDeliveryRepo(t), newTestPrefsRepo(t), sender, nil, nil, resolver, nil, m, application.RetryConfig{})
	h := event.NewHandler(svc)

	producer, err := kafka.NewProducer(testKafkaBrokers, metrics.Noop{})
	require.NoError(t, err)
	t.Cleanup(func() { _ = producer.Close() })

	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()

	consumer := kafka.NewConsumer(testKafkaBrokers, nil, metrics.Noop{})
	errCh := make(chan error, 1)
	topic := kafka.Topic("payment-succeeded-notification-consumer")
	consumerGroup := "notification-consumer-payment-succeeded"
	go func() {
		errCh <- consumer.Subscribe(ctx, topic, consumerGroup, func(_ context.Context, msg []byte) error {
			if herr := h.PaymentSucceeded(ctx, msg); herr != nil {
				t.Logf("handler error: %v", herr)
				return herr
			}
			cancel()
			return nil
		})
	}()

	succeeded := message.PaymentSucceeded{
		PaymentID:   "test-payment-1",
		OrderID:     "test-order-2",
		UserID:      "test-user-2",
		AmountCents: 2500,
		Currency:    "USD",
		ProviderRef: "ref-1",
	}
	require.NoError(t, producer.PublishMessage(t.Context(), topic, succeeded.OrderID, succeeded))

	select {
	case <-ctx.Done():
	case subErr := <-errCh:
		require.NoError(t, subErr)
	case <-time.After(10 * time.Second):
		t.Fatal("timeout waiting for message")
	}

	assert.Equal(t, []string{metrics.StatusSuccess}, m.GetSent())

	stored, err := repo.ListByUserID(t.Context(), "test-user-2", 1, 20)
	require.NoError(t, err)
	require.Len(t, stored, 1, "the consumed event must be recorded exactly once")
	assert.Equal(t, "test-order-2", stored[0].OrderID)
	assert.Equal(t, domain.TypePaymentSucceeded, stored[0].Type)
	assert.Contains(t, stored[0].Body, "R$ 25,00", "the event payload survived the round trip")
}

func TestNotificationConsumer_RespectsOrderUpdatesPreference(t *testing.T) {
	m := &fakeMetrics{}
	repo := newTestRepo(t)
	prefs := newTestPrefsRepo(t)
	sender := newFakeEmailSender()
	resolver := newFakeRecipientResolver()
	svc := application.NewService(repo, newTestDeliveryRepo(t), prefs, sender, nil, nil, resolver, nil, m, application.RetryConfig{})
	h := event.NewHandler(svc)

	userID := uid(t, "muted-user")
	disabled, err := domain.NewPreferences(userID, false, true, true, false)
	require.NoError(t, err)
	require.NoError(t, prefs.Upsert(t.Context(), disabled))

	producer, err := kafka.NewProducer(testKafkaBrokers, metrics.Noop{})
	require.NoError(t, err)
	t.Cleanup(func() { _ = producer.Close() })

	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()

	consumer := kafka.NewConsumer(testKafkaBrokers, nil, metrics.Noop{})
	errCh := make(chan error, 1)
	topic := kafka.Topic("order-placed-notification-muted")
	consumerGroup := "notification-consumer-muted"
	go func() {
		errCh <- consumer.Subscribe(ctx, topic, consumerGroup, func(_ context.Context, msg []byte) error {
			if herr := h.OrderPlaced(ctx, msg); herr != nil {
				t.Logf("handler error: %v", herr)
				return herr
			}
			cancel()
			return nil
		})
	}()

	orderPlaced := message.OrderPlaced{
		OrderID:    uid(t, "muted-order"),
		UserID:     userID,
		TotalCents: 1500,
		ItemCount:  1,
	}
	require.NoError(t, producer.PublishMessage(t.Context(), topic, orderPlaced.OrderID, orderPlaced))

	select {
	case <-ctx.Done():
	case subErr := <-errCh:
		require.NoError(t, subErr)
	case <-time.After(10 * time.Second):
		t.Fatal("timeout waiting for message")
	}

	assert.Empty(t, m.GetSent(), "a disabled category must not send")
	stored, err := repo.ListByUserID(t.Context(), userID, 1, 20)
	require.NoError(t, err)
	assert.Empty(t, stored, "a suppressed notification must not be recorded")
}
