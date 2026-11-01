// Package event holds the catalog domain's message input adapters. A handler here
// is structurally identical to an HTTP controller: it deserializes a transport
// payload (JSON bytes off a Kafka topic) into a call on the application service.
// It knows nothing about Kafka.
package event

import (
	"context"
	"encoding/json"
	"errors"
	"minimart/internal/catalog/application"
	"minimart/internal/catalog/domain"
	"minimart/pkg/kafka"
	"minimart/pkg/message"
	"minimart/pkg/moderation"
)

// ModerateReview is the input adapter for the review-created topic.
// It triggers AI moderation for the review. Idempotent: if the review is
// already moderated, it returns nil without error.
func (h *Handler) ModerateReview(ctx context.Context, msg []byte) error {
	var event message.ReviewCreated
	if err := json.Unmarshal(msg, &event); err != nil {
		return err
	}
	err := h.svc.ModerateReview(ctx, application.ModerateReviewRequest{ReviewID: event.ReviewID})
	switch {
	case err == nil, errors.Is(err, domain.ErrReviewAlreadyModerated):
		return nil
	case errors.Is(err, domain.ErrModerationFailed),
		errors.Is(err, moderation.ErrContentTooLong),
		errors.Is(err, moderation.ErrInvalidResponse),
		errors.Is(err, moderation.ErrProviderDisabled):
		// These cannot succeed on retry: a rejected result or a disabled
		// provider would burn the whole retry budget before reaching the DLQ.
		return kafka.Permanent(err)
	default:
		return err
	}
}
