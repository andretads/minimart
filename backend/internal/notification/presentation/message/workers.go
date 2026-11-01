package message

import (
	"context"
	"encoding/json"
	"errors"
	"minimart/internal/notification/application"
	"minimart/internal/notification/domain"
	"minimart/internal/notification/presentation/message/event"
	"minimart/pkg/kafka"
	"minimart/pkg/worker"
)

func Workers(ctx context.Context, w *worker.Worker, svc *application.Service) {
	h := event.NewHandler(svc)
	w.AttachKafka(ctx, kafka.OrderPlacedTopic, kafka.NotificationConsumer, h.OrderPlaced)
	w.AttachKafka(ctx, kafka.OrderCancelledTopic, kafka.NotificationConsumer, h.OrderCancelled)
	w.AttachKafka(ctx, kafka.PaymentSucceededTopic, kafka.NotificationConsumer, h.PaymentSucceeded)
	w.AttachKafka(ctx, kafka.PaymentFailedTopic, kafka.NotificationConsumer, h.PaymentFailed)
	w.AttachKafka(ctx, kafka.ShipmentCreatedTopic, kafka.NotificationShipmentConsumer, h.ShipmentCreated)
	w.AttachKafka(ctx, kafka.ShipmentShippedTopic, kafka.NotificationShipmentConsumer, h.ShipmentShipped)
	w.AttachKafka(ctx, kafka.ShipmentDeliveredTopic, kafka.NotificationShipmentConsumer, h.ShipmentDelivered)

	handleDelivery := deliveryHandler(svc)
	w.AttachKafka(ctx, kafka.NotificationEmailTopic, kafka.NotificationEmailConsumer, handleDelivery)
	w.AttachKafka(ctx, kafka.NotificationSMSTopic, kafka.NotificationSMSConsumer, handleDelivery)
	w.AttachKafka(ctx, kafka.NotificationPushTopic, kafka.NotificationPushConsumer, handleDelivery)
}

func deliveryHandler(svc *application.Service) func(context.Context, []byte) error {
	return func(ctx context.Context, msg []byte) error {
		deliveryID := decodeDeliveryID(msg)
		if deliveryID == "" {
			return kafka.Permanent(domain.ErrEmptyID)
		}
		err := svc.DeliverNotification(ctx, deliveryID)
		if errors.Is(err, domain.ErrNotificationNotFound) || errors.Is(err, domain.ErrDeliveryNotFound) {
			return kafka.Permanent(err)
		}
		return err
	}
}

func decodeDeliveryID(msg []byte) string {
	var deliveryID string
	if err := json.Unmarshal(msg, &deliveryID); err != nil {
		return string(msg)
	}
	return deliveryID
}
