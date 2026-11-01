package sms

import (
	"context"
	"minimart/internal/notification/domain"
)

type LogSender struct{}

func NewLog() *LogSender { return &LogSender{} }

func (s *LogSender) Send(ctx context.Context, m domain.SMS) (domain.Receipt, error) {
	return domain.Receipt{ProviderMessageID: "log-" + m.To, Delivered: false}, nil
}
