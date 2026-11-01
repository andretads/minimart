package push

import (
	"context"
	"minimart/internal/notification/domain"
)

type LogSender struct{}

func NewLog() *LogSender { return &LogSender{} }

func (s *LogSender) Send(ctx context.Context, p domain.Push) (domain.Receipt, error) {
	return domain.Receipt{ProviderMessageID: "log-" + p.Token, Delivered: false}, nil
}
