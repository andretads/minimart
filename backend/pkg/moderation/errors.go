package moderation

import "errors"

var (
	ErrProviderUnavailable = errors.New("moderation provider unavailable")
	ErrInvalidResponse     = errors.New("invalid moderation response")
	ErrTimeout             = errors.New("moderation request timeout")
	ErrContentTooLong      = errors.New("content exceeds maximum length")
	ErrProviderDisabled    = errors.New("moderation provider disabled")
)
