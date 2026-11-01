// Package catalog wires the catalog domain into the modular monolith. It is the
// domain's composition root: the one place that names concrete adapters.
package catalog

import (
	"context"
	"errors"
	"minimart/conf"
	"minimart/internal/catalog/application"
	"minimart/internal/catalog/domain"
	"minimart/internal/catalog/infrastructure/postgres"
	"minimart/pkg/metrics"
	"minimart/pkg/moderation"
	"minimart/pkg/moderation/llamaguard"
	"minimart/pkg/module"
	"minimart/pkg/module/contracts"
	"minimart/pkg/outbox"
	outboxpg "minimart/pkg/outbox/postgres"
	"minimart/pkg/storage/s3"
	"minimart/pkg/worker"
	"net/http"

	"github.com/jackc/pgx/v5/pgxpool"

	goredis "github.com/redis/go-redis/v9"

	cataloghttp "minimart/internal/catalog/presentation/http"
	catalogmessage "minimart/internal/catalog/presentation/message"
	catalogmodule "minimart/internal/catalog/presentation/module"
)

// Module is the catalog domain's self-wiring unit.
type Module struct {
	registry           *contracts.Contracts
	readPool           *pgxpool.Pool
	writePool          *pgxpool.Pool
	cache              *goredis.Client
	m                  metrics.Metrics
	svc                *application.Service
	handlers           *cataloghttp.Handlers
	s3Client           *s3.Client
	repo               domain.Repository
	outboxStore        *outboxpg.Store
	moderationProvider moderation.Provider
}

var _ module.Module = (*Module)(nil)

func NewModule(registry *contracts.Contracts, readPool, writePool *pgxpool.Pool, m metrics.Metrics, cache ...*goredis.Client) *Module {
	mod := &Module{registry: registry, readPool: readPool, writePool: writePool, m: m}
	if len(cache) > 0 {
		mod.cache = cache[0]
	}
	return mod
}

func (m *Module) Name() string { return "catalog" }

// Init chooses the concrete storage adapter. Swapping storage is this one line;
// no other layer is touched. (Earlier stages used memory.New() here.)
func (m *Module) Init(ctx context.Context) error {
	// Initialize outbox store for publishing review-created events
	m.outboxStore = outboxpg.NewStore(m.writePool)

	// Initialize repository with outbox publisher
	outboxPublisher := outbox.NewPublisher(m.outboxStore)
	repo := postgres.NewCatalogRepository(m.readPool, m.writePool, outboxPublisher, m.cache)
	m.repo = repo

	// Initialize S3 client from conf
	s3Cfg := s3.Config{
		Endpoint:       conf.S3Endpoint(),
		Bucket:         conf.S3Bucket(),
		Region:         conf.S3Region(),
		AccessKey:      conf.S3AccessKey(),
		SecretKey:      conf.S3SecretKey(),
		UsePathStyle:   conf.S3UsePathStyle(),
		PublicEndpoint: conf.S3PublicEndpoint(),
	}
	s3Client, err := s3.NewClient(s3Cfg)
	if err != nil {
		return err
	}
	m.s3Client = s3Client

	// Initialize moderation provider
	if conf.ModerationEnabled() {
		cfg := llamaguard.Config{
			Endpoint: conf.ModerationEndpoint(),
			Model:    conf.ModerationModel(),
			Timeout:  conf.ModerationTimeout(),
		}
		provider, err := llamaguard.NewClient(cfg)
		if err != nil {
			return err
		}
		m.moderationProvider = provider
	} else {
		m.moderationProvider = moderation.NewNoopProvider()
	}

	return nil
}

func (m *Module) RegisterContracts(ctx context.Context, c *contracts.Contracts) error {
	// Category contract is available via registry (category module registers before catalog)
	categoryProvider := m.registry.CategoryClient
	m.svc = application.NewService(m.repo, m.s3Client, categoryProvider, conf.S3PresignTTL(), int64(conf.S3MaxUploadSizeMB())*1024*1024, m.m, m.moderationProvider)
	c.CatalogClient = catalogmodule.New(m.svc)
	return nil
}

func (m *Module) RegisterHTTP(ctx context.Context, mux *http.ServeMux) error {
	// The customer contract is published during the contracts phase, which
	// completes before any module registers its routes, so it is safe to
	// resolve the authenticated caller here.
	customerClient := m.registry.CustomerClient
	if customerClient == nil {
		return errors.New("catalog: customer contract not registered")
	}
	m.handlers = cataloghttp.NewHandlers(m.svc, customerClient)
	m.handlers.Register(mux)
	return nil
}

// RegisterWorkers wires the catalog domain's Kafka consumer: it releases an
// order's stock reservation when the order will not be fulfilled.
func (m *Module) RegisterWorkers(ctx context.Context, w *worker.Worker) error {
	catalogmessage.Workers(ctx, w, m.svc)
	return nil
}
