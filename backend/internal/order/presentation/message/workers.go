package message

import (
	"context"
	"minimart/internal/order/application"
	"minimart/internal/order/presentation/message/event"
	"minimart/pkg/kafka"
	"minimart/pkg/worker"
)

// Workers registers the order domain's Kafka consumers onto the worker: its own
// order-placed confirmation, and the terminal payment outcomes that drive the
// order's status.
func Workers(ctx context.Context, w *worker.Worker, svc *application.Service) {
	h := event.NewHandler(svc)
	w.AttachKafka(ctx, kafka.OrderPlacedTopic, kafka.OrderConfirmationConsumer, h.OrderConfirmation)
	w.AttachKafka(ctx, kafka.PaymentSucceededTopic, kafka.OrderPaymentConsumer, h.PaymentSucceeded)
	w.AttachKafka(ctx, kafka.PaymentFailedTopic, kafka.OrderPaymentConsumer, h.PaymentFailed)
	w.AttachKafka(ctx, kafka.PaymentUnknownTopic, kafka.OrderPaymentConsumer, h.PaymentUnknown)
}
