package resend

import (
	"context"
	"fmt"
	"minimart/internal/notification/domain"

	"github.com/resend/resend-go/v4"
)

type Sender struct {
	client  *resend.Client
	from    string
	replyTo string
}

func New(apiKey, from, replyTo string) *Sender {
	return &Sender{
		client:  resend.NewClient(apiKey),
		from:    from,
		replyTo: replyTo,
	}
}

func (s *Sender) Send(ctx context.Context, e domain.Email) (domain.Receipt, error) {
	params := &resend.SendEmailRequest{
		From:    s.from,
		To:      []string{e.To},
		Subject: e.Subject,
		Text:    e.Text,
		Html:    e.HTML,
	}
	if s.replyTo != "" {
		params.ReplyTo = s.replyTo
	}

	var err error
	if e.IdempotencyKey != "" {
		_, err = s.client.Emails.SendWithOptions(ctx, params, &resend.SendEmailOptions{
			IdempotencyKey: e.IdempotencyKey,
		})
	} else {
		_, err = s.client.Emails.SendWithContext(ctx, params)
	}
	if err != nil {
		return domain.Receipt{}, fmt.Errorf("resend send email: %w", err)
	}
	return domain.Receipt{ProviderMessageID: "resend", Delivered: false}, nil
}
