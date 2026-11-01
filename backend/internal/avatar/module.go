package avatar

import (
	"context"
	"minimart/conf"
	"minimart/internal/avatar/application"
	"minimart/internal/avatar/domain"
	"minimart/internal/avatar/infrastructure/postgres"
	avatarhttp "minimart/internal/avatar/presentation/http"
	"minimart/pkg/metrics"
	"minimart/pkg/module"
	"minimart/pkg/module/contracts"
	"minimart/pkg/storage/s3"
	"minimart/pkg/worker"
	"net/http"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Module is the avatar domain's self-wiring unit.
type Module struct {
	readPool  *pgxpool.Pool
	writePool *pgxpool.Pool
	m         metrics.Metrics
	svc       *application.Service
	handlers  *avatarhttp.Handlers
	s3Client  *s3.Client
	repo      domain.Repository
}

var _ module.Module = (*Module)(nil)

func NewModule(readPool, writePool *pgxpool.Pool, m metrics.Metrics) *Module {
	return &Module{readPool: readPool, writePool: writePool, m: m}
}

func (m *Module) Name() string { return "avatar" }

// Init chooses the concrete storage adapter and initializes the S3 client.
func (m *Module) Init(ctx context.Context) error {
	repo := postgres.NewAvatarRepository(m.readPool, m.writePool)
	m.repo = repo

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

	m.svc = application.NewService(m.repo, m.s3Client, conf.S3PresignTTL(), int64(conf.S3MaxUploadSizeMB())*1024*1024, m.m)
	m.handlers = avatarhttp.NewHandlers(m.svc)
	return nil
}

func (m *Module) RegisterContracts(ctx context.Context, c *contracts.Contracts) error {
	// Avatar publishes no contract.
	return nil
}

func (m *Module) RegisterHTTP(ctx context.Context, mux *http.ServeMux) error {
	m.handlers.Register(mux)
	return nil
}

func (m *Module) RegisterWorkers(ctx context.Context, w *worker.Worker) error {
	// Avatar has no background workers.
	return nil
}
