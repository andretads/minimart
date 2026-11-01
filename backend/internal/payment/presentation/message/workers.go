// Package message wires the payment domain's Kafka consumers.
package message

import (
	"context"
	"minimart/internal/payment/application"
	"minimart/internal/payment/presentation/message/event"
	"minimart/pkg/kafka"
	"minimart/pkg/worker"
)

// Workers attaches the payment domain's Kafka consumers: the order-cancelled
// topic, whose events trigger refunds of the order's captured payments.
func Workers(ctx context.Context, w *worker.Worker, svc *application.Service) {
	h := event.NewHandler(svc)
	w.AttachKafka(ctx, kafka.OrderCancelledTopic, kafka.PaymentRefundConsumer, h.OrderCancelled)
}
