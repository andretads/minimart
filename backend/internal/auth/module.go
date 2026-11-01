package auth

import (
	"context"
	"minimart/internal/auth/application"
	"minimart/internal/auth/infrastructure/postgres"
	authhttp "minimart/internal/auth/presentation/http"
	authmodule "minimart/internal/auth/presentation/module"
	"minimart/pkg/auth"
	"minimart/pkg/metrics"
	"minimart/pkg/module"
	"minimart/pkg/module/contracts"
	"minimart/pkg/worker"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Module struct {
	registry             *contracts.Contracts
	readPool             *pgxpool.Pool
	writePool            *pgxpool.Pool
	tokens               *auth.TokenService
	cookieEncryption     *auth.CookieEncryption
	cookieConfig         auth.CookieConfig
	rememberMeRefreshTTL time.Duration
	m                    metrics.Metrics
	svc                  *application.Service
	handlers             *authhttp.Handlers
}

var _ module.Module = (*Module)(nil)

func NewModule(registry *contracts.Contracts, readPool, writePool *pgxpool.Pool, tokens *auth.TokenService, cookieEncryption *auth.CookieEncryption, cookieConfig auth.CookieConfig, rememberMeRefreshTTL time.Duration, m metrics.Metrics) *Module {
	return &Module{
		registry:             registry,
		readPool:             readPool,
		writePool:            writePool,
		tokens:               tokens,
		cookieEncryption:     cookieEncryption,
		cookieConfig:         cookieConfig,
		rememberMeRefreshTTL: rememberMeRefreshTTL,
		m:                    m,
	}
}

func (m *Module) Name() string { return "auth" }

func (m *Module) Init(ctx context.Context) error {
	repo := postgres.NewRepository(m.readPool, m.writePool)
	m.svc = application.NewService(repo, m.registry, m.m)
	m.handlers = authhttp.NewHandlers(m.svc, m.tokens, m.cookieEncryption, m.cookieConfig, m.rememberMeRefreshTTL)
	return nil
}

func (m *Module) RegisterContracts(ctx context.Context, c *contracts.Contracts) error {
	c.AuthClient = authmodule.New(m.svc)
	return nil
}

func (m *Module) RegisterHTTP(ctx context.Context, mux *http.ServeMux) error {
	m.handlers.Register(mux)
	return nil
}

func (m *Module) RegisterWorkers(ctx context.Context, w *worker.Worker) error {
	return nil
}

// EnsureAdmins ensures the admin role is assigned to the given usernames.
func (m *Module) EnsureAdmins(ctx context.Context, usernames []string) error {
	return m.svc.EnsureAdmins(ctx, usernames)
}

// PermissionsForAccount returns the permissions for the given account.
func (m *Module) PermissionsForAccount(ctx context.Context, accountID string) ([]string, error) {
	return m.svc.PermissionsForAccount(ctx, accountID)
}

// Service returns the application service for external consumers (e.g., authz middleware).
func (m *Module) Service() *application.Service {
	return m.svc
}
