package application

import (
	"context"
	"fmt"
	"minimart/pkg/kafka"
	"minimart/pkg/logger"
)

func (s *Service) DispatchDueDeliveries(ctx context.Context, workerID string, limit int) (int, error) {
	if limit <= 0 {
		limit = 50
	}
	deliveries, err := s.deliveries.ClaimDue(ctx, workerID, limit)
	if err != nil {
		return 0, err
	}
	published := 0
	var lastErr error
	for _, d := range deliveries {
		topic := channelTopic(d.Channel)
		if err := s.publisher.Publish(ctx, kafka.Message{
			Topic:   topic,
			Key:     d.ID,
			Payload: []byte(d.ID),
		}); err != nil {
			lastErr = fmt.Errorf("publish delivery %s: %w", d.ID, err)
			_ = s.deliveries.ReleaseLease(ctx, d.ID, workerID)
			logger.FromContext(ctx).Error("failed to dispatch delivery", "delivery_id", d.ID, "error", err)
			continue
		}
		published++
	}
	return published, lastErr
}
