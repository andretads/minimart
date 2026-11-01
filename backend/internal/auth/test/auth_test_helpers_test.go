//go:build integration

package test

import (
	"bytes"
	"context"
	"crypto/rand"
	"errors"
	"log/slog"
	"minimart/internal/auth/application"
	authdomain "minimart/internal/auth/domain"
	authpg "minimart/internal/auth/infrastructure/postgres"
	customerapp "minimart/internal/customer/application"
	customerpg "minimart/internal/customer/infrastructure/postgres"
	"minimart/internal/customer/presentation/module/client"
	"minimart/pkg/metrics"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"
)

func mustPools(t *testing.T) (*pgxpool.Pool, *pgxpool.Pool) {
	t.Helper()
	ctx := t.Context()
	readPool, err := newPool(ctx, testDSN)
	require.NoError(t, err)
	writePool, err := newPool(ctx, testDSN)
	require.NoError(t, err)
	return readPool, writePool
}

func newPool(ctx context.Context, dsn string) (*pgxpool.Pool, error) {
	return pgxpool.New(ctx, dsn)
}

func newAuthService(t *testing.T, prov application.CustomerProvisioner) (*application.Service, *authpg.Repository) {
	t.Helper()
	readPool, writePool := mustPools(t)
	repo := authpg.NewRepository(readPool, writePool)
	return application.NewService(repo, prov, metrics.Noop{}), repo
}

func newFailingAuthService(t *testing.T) *application.Service {
	t.Helper()
	readPool, writePool := mustPools(t)
	repo := authpg.NewRepository(readPool, writePool)
	return application.NewService(&failingSaveRepo{Repository: repo}, nil, metrics.Noop{})
}

func newRealCustomerProvisioner(t *testing.T) application.CustomerProvisioner {
	t.Helper()
	readPool, writePool := mustPools(t)
	return customerProvisioner{
		svc: customerapp.NewService(customerpg.NewCustomerRepository(readPool, writePool), metrics.Noop{}),
	}
}

func uniqueUsername(prefix string) string {
	return prefix + "-" + rand.Text()
}

func validSignUpRequest(username string) application.SignUpRequest {
	return application.SignUpRequest{
		Username: username,
		Password: "secret123",
		Email:    username + "@example.com",
	}
}

func newTestAccount(t *testing.T, svc *application.Service, username string) authdomain.Account {
	t.Helper()
	acc, err := svc.SignUp(t.Context(), validSignUpRequest(username))
	require.NoError(t, err)
	return acc
}

func roleIDs(roles []authdomain.Role) []string {
	ids := make([]string, len(roles))
	for i, role := range roles {
		ids[i] = role.ID
	}
	return ids
}

func countRole(roles []authdomain.Role, roleID string) int {
	count := 0
	for _, role := range roles {
		if role.ID == roleID {
			count++
		}
	}
	return count
}

// captureLogs redirects the default slog logger into a buffer for the duration
// of the test, so best-effort warnings can be asserted.
func captureLogs(t *testing.T) *bytes.Buffer {
	t.Helper()
	var buf bytes.Buffer
	previous := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(&buf, nil)))
	t.Cleanup(func() { slog.SetDefault(previous) })
	return &buf
}

type customerProvisioner struct {
	svc *customerapp.Service
}

func (p customerProvisioner) EnsureCustomer(ctx context.Context, userID string) (client.Customer, error) {
	c, err := p.svc.EnsureCustomer(ctx, userID)
	if err != nil {
		return client.Customer{}, err
	}
	return client.Customer{
		ID:          c.ID,
		UserID:      c.UserID,
		DisplayName: c.DisplayName,
		Status:      string(c.Status),
		CreatedAt:   c.CreatedAt,
	}, nil
}

type fakeCustomerProvisioner struct {
	err error
}

func (f *fakeCustomerProvisioner) EnsureCustomer(_ context.Context, userID string) (client.Customer, error) {
	if f.err != nil {
		return client.Customer{}, f.err
	}
	return client.Customer{ID: "cust-" + userID, UserID: userID}, nil
}

// failingSaveRepo injects a deterministic storage failure while delegating all
// other operations to the real Postgres repository.
type failingSaveRepo struct {
	*authpg.Repository
}

func (r *failingSaveRepo) Save(context.Context, authdomain.Account) error {
	return errors.New("disk on fire")
}
