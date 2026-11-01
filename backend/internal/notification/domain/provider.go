package domain

import (
	"context"
	"time"
)

type Receipt struct {
	ProviderMessageID string
	Delivered         bool
	DeliveredAt       *time.Time
}

type ProviderError struct {
	Transient  bool
	Permanent  bool
	RetryAfter *time.Duration
	Err        error
}

func (e ProviderError) Error() string {
	if e.Err != nil {
		return e.Err.Error()
	}
	return "provider error"
}

func (e ProviderError) Unwrap() error { return e.Err }

func (e ProviderError) IsTransient() bool { return e.Transient && !e.Permanent }

type SMS struct {
	To             string
	Text           string
	IdempotencyKey string
}

type Push struct {
	Token          string
	Title          string
	Body           string
	IdempotencyKey string
}

type EmailSender interface {
	Send(ctx context.Context, e Email) (Receipt, error)
}

type SMSSender interface {
	Send(ctx context.Context, m SMS) (Receipt, error)
}

type PushSender interface {
	Send(ctx context.Context, p Push) (Receipt, error)
}

type RecipientResolver interface {
	EmailForUser(ctx context.Context, userID string) (string, error)
	PhoneForUser(ctx context.Context, userID string) (string, error)
	PushTokenForUser(ctx context.Context, userID string) (string, error)
}
