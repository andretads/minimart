// Package showcase wires the showcase domain into the modular monolith.
// It is the domain's composition root: the one place that names concrete adapters.
package showcase

import (
	"context"
	"minimart/conf"
	"minimart/internal/showcase/application"
	"minimart/internal/showcase/infrastructure/postgres"
	"minimart/pkg/module"
	"minimart/pkg/module/contracts"
	"minimart/pkg/storage/s3"
	"minimart/pkg/worker"
	"net/http"

	"github.com/jackc/pgx/v5/pgxpool"

	showcasehttp "minimart/internal/showcase/presentation/http"
)

// Module is the showcase domain's self-wiring unit.
type Module struct {
	// readPool is the Postgres replica pool. The showcase domain is read-only
	// and never uses the primary write pool.
	readPool *pgxpool.Pool

	// svc is the application service orchestrating use cases.
	svc *application.Service

	// handlers are the HTTP handlers for the public showcase endpoints.
	handlers *showcasehttp.Handlers
}

var _ module.Module = (*Module)(nil)

// NewModule constructs the showcase module with the given read pool.
func NewModule(readPool *pgxpool.Pool) *Module {
	return &Module{readPool: readPool}
}

// Name returns the module name used in logging and metrics.
func (m *Module) Name() string { return "showcase" }

// Init chooses the concrete storage adapter and builds the service and handlers.
func (m *Module) Init(ctx context.Context) error {
	repo := postgres.NewRepository(m.readPool)

	s3Client, err := s3.NewClient(s3.Config{
		Endpoint:       conf.S3Endpoint(),
		Bucket:         conf.S3Bucket(),
		Region:         conf.S3Region(),
		AccessKey:      conf.S3AccessKey(),
		SecretKey:      conf.S3SecretKey(),
		UsePathStyle:   conf.S3UsePathStyle(),
		PublicEndpoint: conf.S3PublicEndpoint(),
	})
	if err != nil {
		return err
	}

	m.svc = application.NewService(repo, s3Client, conf.S3PresignTTL())
	m.handlers = showcasehttp.NewHandlers(m.svc)
	return nil
}

// RegisterContracts publishes no contract. The showcase domain is a public
// read-only display layer and does not expose a cross-domain contract.
func (m *Module) RegisterContracts(ctx context.Context, c *contracts.Contracts) error {
	return nil
}

// RegisterHTTP registers the public showcase routes on the given ServeMux.
func (m *Module) RegisterHTTP(ctx context.Context, mux *http.ServeMux) error {
	m.handlers.Register(mux)
	return nil
}

// RegisterWorkers is a no-op. The showcase domain has no background consumers
// and no outbox relay; it runs only in the server role.
func (m *Module) RegisterWorkers(ctx context.Context, w *worker.Worker) error {
	return nil
}
