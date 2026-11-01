package application

import (
	"minimart/internal/notification/domain"
	"minimart/pkg/kafka"
	"minimart/pkg/metrics"
	"time"
)

type Service struct {
	repo       domain.NotificationRepository
	deliveries domain.DeliveryRepository
	prefs      domain.PreferencesRepository
	sender     domain.EmailSender
	smsSender  domain.SMSSender
	pushSender domain.PushSender
	recipients domain.RecipientResolver
	publisher  kafka.HeaderPublisher
	m          metrics.Metrics
	retry      RetryConfig
}

type RetryConfig struct {
	MaxAttempts int
	BaseDelay   time.Duration
	MaxDelay    time.Duration
}

func NewService(repo domain.NotificationRepository, deliveries domain.DeliveryRepository, prefs domain.PreferencesRepository, sender domain.EmailSender, smsSender domain.SMSSender, pushSender domain.PushSender, recipients domain.RecipientResolver, publisher kafka.HeaderPublisher, m metrics.Metrics, retry RetryConfig) *Service {
	if retry.MaxAttempts < 1 {
		retry.MaxAttempts = 5
	}
	if retry.BaseDelay <= 0 {
		retry.BaseDelay = time.Second
	}
	if retry.MaxDelay < retry.BaseDelay {
		retry.MaxDelay = 30 * time.Second
	}
	return &Service{repo: repo, deliveries: deliveries, prefs: prefs, sender: sender, smsSender: smsSender, pushSender: pushSender, recipients: recipients, publisher: publisher, m: m, retry: retry}
}
