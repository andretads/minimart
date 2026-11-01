package log

import (
	"context"
	"log/slog"
	"minimart/internal/notification/domain"
)

type Sender struct{}

func New() *Sender { return &Sender{} }

func (s *Sender) Send(ctx context.Context, e domain.Email) (domain.Receipt, error) {
	slog.Info("email not sent (resend not configured)", "to", e.To, "subject", e.Subject)
	return domain.Receipt{ProviderMessageID: "log", Delivered: false}, nil
}
