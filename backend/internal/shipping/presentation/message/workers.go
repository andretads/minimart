package message

import (
	"context"
	"minimart/internal/shipping/application"
	"minimart/internal/shipping/presentation/message/event"
	"minimart/pkg/kafka"
	"minimart/pkg/worker"
)

func Workers(ctx context.Context, w *worker.Worker, svc *application.Service) {
	h := event.NewHandler(svc)
	w.AttachKafka(ctx, kafka.OrderPaidTopic, kafka.ShippingShipmentConsumer, h.OrderPaid)
}
