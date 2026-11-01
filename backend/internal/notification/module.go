package notification

import (
	"context"
	"minimart/internal/notification/application"
	"minimart/internal/notification/domain"
	notificationmodule "minimart/internal/notification/infrastructure/client"
	"minimart/internal/notification/infrastructure/email/log"
	"minimart/internal/notification/infrastructure/mongo"
	"minimart/internal/notification/infrastructure/push"
	notificationrelay "minimart/internal/notification/infrastructure/relay"
	"minimart/internal/notification/infrastructure/sms"
	notificationhttp "minimart/internal/notification/presentation/http"
	"minimart/internal/notification/presentation/message"
	"minimart/pkg/kafka"
	"minimart/pkg/metrics"
	"minimart/pkg/module/contracts"
	"minimart/pkg/worker"
	"net/http"
	"time"

	mongodriver "go.mongodb.org/mongo-driver/v2/mongo"
)

type Module struct {
	db        *mongodriver.Database
	registry  *contracts.Contracts
	sender    domain.EmailSender
	sms       domain.SMSSender
	push      domain.PushSender
	m         metrics.Metrics
	publisher kafka.HeaderPublisher
	svc       *application.Service
	handlers  *notificationhttp.Handlers
	relay     *notificationrelay.Relay
}

func NewModule(db *mongodriver.Database, registry *contracts.Contracts, sender domain.EmailSender, smsSender domain.SMSSender, pushSender domain.PushSender, publisher kafka.HeaderPublisher, m metrics.Metrics) *Module {
	return &Module{db: db, registry: registry, sender: sender, sms: smsSender, push: pushSender, publisher: publisher, m: m}
}

func (m *Module) Name() string { return "notification" }

func (m *Module) Init(ctx context.Context) error {
	repo := mongo.NewNotificationRepository(m.db)
	if err := repo.EnsureIndexes(ctx); err != nil {
		return err
	}
	deliveries := mongo.NewDeliveryRepository(m.db)
	if err := deliveries.EnsureIndexes(ctx); err != nil {
		return err
	}
	prefs := mongo.NewPreferencesRepository(m.db)

	recipients := notificationmodule.NewAccountReader(m.registry)

	sender := m.sender
	if sender == nil {
		sender = log.New()
	}
	smsSender := m.sms
	if smsSender == nil {
		smsSender = sms.NewLog()
	}
	pushSender := m.push
	if pushSender == nil {
		pushSender = push.NewLog()
	}

	m.svc = application.NewService(repo, deliveries, prefs, sender, smsSender, pushSender, recipients, m.publisher, m.m, application.RetryConfig{
		MaxAttempts: 5,
		BaseDelay:   time.Second,
		MaxDelay:    30 * time.Second,
	})
	m.handlers = notificationhttp.NewHandlers(m.svc)
	m.relay = notificationrelay.New(m.svc)
	return nil
}

func (m *Module) RegisterContracts(ctx context.Context, c *contracts.Contracts) error { return nil }

func (m *Module) RegisterHTTP(ctx context.Context, mux *http.ServeMux) error {
	m.handlers.Register(mux)
	return nil
}

func (m *Module) RegisterWorkers(ctx context.Context, w *worker.Worker) error {
	message.Workers(ctx, w, m.svc)
	if m.relay != nil && m.publisher != nil {
		w.AttachBackground(ctx, m.relay.Run)
	}
	return nil
}
