//go:build integration

package test

import (
	"context"
	"encoding/json"
	"minimart/internal/notification/application"
	"minimart/internal/notification/domain"
	"minimart/internal/notification/presentation/message/event"
	"minimart/pkg/kafka"
	"minimart/pkg/message"
	"minimart/pkg/metrics"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func mustJSON(t *testing.T, payload any) []byte {
	t.Helper()
	body, err := json.Marshal(payload)
	require.NoError(t, err)
	return body
}

func TestNotificationShipmentConsumer_ConsumesShipmentEvents(t *testing.T) {
	m := &fakeMetrics{}
	repo := newTestRepo(t)
	sender := newFakeEmailSender()
	resolver := newFakeRecipientResolver()
	orderID := uid(t, "shipment-order")
	userID := uid(t, "shipment-user")
	resolver.SetEmail(userID, "user1@example.com")
	svc := application.NewService(repo, newTestDeliveryRepo(t), newTestPrefsRepo(t), sender, nil, nil, resolver, nil, m, application.RetryConfig{})
	h := event.NewHandler(svc)

	producer, err := kafka.NewProducer(testKafkaBrokers, metrics.Noop{})
	require.NoError(t, err)
	t.Cleanup(func() { _ = producer.Close() })

	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()

	consumer := kafka.NewConsumer(testKafkaBrokers, nil, metrics.Noop{})
	errCh := make(chan error, 1)
	consumerGroup := "notification-shipment-consumer-" + uid(t, "group")
	go func() {
		errCh <- consumer.Subscribe(ctx, kafka.ShipmentCreatedTopic, consumerGroup, func(_ context.Context, msg []byte) error {
			if herr := h.ShipmentCreated(ctx, msg); herr != nil {
				t.Logf("handler error: %v", herr)
				return herr
			}
			cancel()
			return nil
		})
	}()

	created := message.ShipmentCreated{
		ShipmentID:        uid(t, "shipment"),
		OrderID:           orderID,
		UserID:            userID,
		MethodID:          uid(t, "method"),
		ShippingCostCents: 599,
		CreatedAt:         "2024-01-01T00:00:00Z",
	}
	require.NoError(t, producer.PublishMessage(t.Context(), kafka.ShipmentCreatedTopic, created.ShipmentID, created))

	select {
	case <-ctx.Done():
	case subErr := <-errCh:
		require.NoError(t, subErr)
	case <-time.After(15 * time.Second):
		t.Fatal("timeout waiting for the shipment-created message")
	}

	stored, err := repo.GetByOrderIDAndType(t.Context(), orderID, domain.TypeShipmentCreated)
	require.NoError(t, err)
	assert.Equal(t, orderID, stored.OrderID)
	assert.Equal(t, userID, stored.UserID)
	assert.Contains(t, stored.Body, "R$ 5,99")
}

func TestNotificationShipmentConsumer_RecordsEveryLifecycleStep(t *testing.T) {
	m := &fakeMetrics{}
	repo := newTestRepo(t)
	sender := newFakeEmailSender()
	resolver := newFakeRecipientResolver()
	orderID := uid(t, "lifecycle-order")
	userID := uid(t, "lifecycle-user")
	resolver.SetEmail(userID, "user1@example.com")
	svc := application.NewService(repo, newTestDeliveryRepo(t), newTestPrefsRepo(t), sender, nil, nil, resolver, nil, m, application.RetryConfig{})
	h := event.NewHandler(svc)

	shipmentID := uid(t, "lifecycle-shipment")
	require.NoError(t, h.ShipmentCreated(t.Context(), mustJSON(t, message.ShipmentCreated{
		ShipmentID:        shipmentID,
		OrderID:           orderID,
		UserID:            userID,
		MethodID:          uid(t, "method"),
		ShippingCostCents: 599,
		CreatedAt:         "2024-01-01T00:00:00Z",
	})))
	require.NoError(t, h.ShipmentShipped(t.Context(), mustJSON(t, message.ShipmentShipped{
		ShipmentID:   shipmentID,
		OrderID:      orderID,
		UserID:       userID,
		TrackingCode: "TRACK123",
		Carrier:      "Correios",
		ShippedAt:    "2024-01-02T00:00:00Z",
	})))
	require.NoError(t, h.ShipmentDelivered(t.Context(), mustJSON(t, message.ShipmentDelivered{
		ShipmentID:  shipmentID,
		OrderID:     orderID,
		UserID:      userID,
		DeliveredAt: "2024-01-03T00:00:00Z",
	})))

	stored, err := repo.ListByUserID(t.Context(), userID, 1, 20)
	require.NoError(t, err)
	require.Len(t, stored, 3, "each lifecycle step records its own notification")

	types := make([]domain.Type, 0, len(stored))
	for _, n := range stored {
		types = append(types, n.Type)
	}
	assert.ElementsMatch(t, []domain.Type{
		domain.TypeShipmentCreated,
		domain.TypeShipmentShipped,
		domain.TypeShipmentDelivered,
	}, types)
}
