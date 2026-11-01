package event

import (
	"context"
	"encoding/json"
	"errors"
	"minimart/internal/catalog/domain"
	"minimart/pkg/kafka"
	"minimart/pkg/message"
)

// AggregateRating updates the rating summary when a review is moderated.
func (h *Handler) AggregateRating(ctx context.Context, msg []byte) error {
	var event message.ReviewModerated
	if err := json.Unmarshal(msg, &event); err != nil {
		return err
	}
	err := h.svc.UpdateRatingSummary(ctx, event.ProductID)
	switch {
	case err == nil:
		return nil
	case errors.Is(err, domain.ErrProductNotFound):
		return kafka.Permanent(err)
	default:
		return err
	}
}
