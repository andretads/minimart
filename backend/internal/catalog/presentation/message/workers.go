package message

import (
	"context"
	"minimart/internal/catalog/application"
	"minimart/internal/catalog/presentation/message/event"
	"minimart/pkg/kafka"
	"minimart/pkg/worker"
)

// Workers registers the catalog domain's Kafka consumers onto the worker: it
// releases an order's stock reservation when the order will not be fulfilled,
// and moderates product reviews.
func Workers(ctx context.Context, w *worker.Worker, svc *application.Service) {
	h := event.NewHandler(svc)
	w.AttachKafka(ctx, kafka.OrderStockReleaseTopic, kafka.CatalogStockConsumer, h.OrderStockRelease)
	w.AttachKafka(ctx, kafka.ReviewCreatedTopic, kafka.ReviewModerationConsumer, h.ModerateReview)
	w.AttachKafka(ctx, kafka.ReviewModeratedTopic, kafka.ReviewRatingAggregatorConsumer, h.AggregateRating)
}
