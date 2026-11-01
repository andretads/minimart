package application

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"minimart/internal/auth/domain"
	"minimart/internal/auth/infrastructure/memory"
	"minimart/internal/customer/presentation/module/client"
	"minimart/pkg/metrics"
	"testing"

	"github.com/stretchr/testify/require"
)

type failingProvisioner struct{}

func (f *failingProvisioner) EnsureCustomer(context.Context, string) (client.Customer, error) {
	return client.Customer{}, errors.New("provisioner unavailable")
}

// failingSaveRepo returns a fixed error on Save so storage failures surface
// without coupling the test to the in-memory repository internals.
type failingSaveRepo struct {
	*memory.Repository
}

func (r *failingSaveRepo) Save(context.Context, domain.Account) error {
	return errors.New("disk on fire")
}

func newTestService(t *testing.T, prov CustomerProvisioner) (*Service, *memory.Repository) {
	t.Helper()
	repo := memory.New()
	return NewService(repo, prov, metrics.Noop{}), repo
}

func seedStandardRoles(t *testing.T, repo *memory.Repository) {
	t.Helper()
	for _, role := range []domain.Role{
		{ID: "customer", Description: "Default role"},
		{ID: "support", Description: "Support staff"},
		{ID: "admin", Description: "Administrator"},
	} {
		require.NoError(t, repo.CreateRole(t.Context(), role))
	}
}

func seedPermissions(t *testing.T, repo *memory.Repository, roleID string, permissionIDs ...string) {
	t.Helper()
	for _, id := range permissionIDs {
		require.NoError(t, repo.CreatePermission(t.Context(), domain.Permission{ID: id}))
		require.NoError(t, repo.AddRolePermission(t.Context(), roleID, id))
	}
}

func validSignUpRequest(username string) SignUpRequest {
	return SignUpRequest{
		Username: username,
		Password: "secret123",
		Email:    username + "@example.com",
	}
}

func createTestAccount(t *testing.T, svc *Service, username string) domain.Account {
	t.Helper()
	acc, err := svc.SignUp(t.Context(), validSignUpRequest(username))
	require.NoError(t, err)
	return acc
}

func roleIDs(roles []domain.Role) []string {
	ids := make([]string, len(roles))
	for i, role := range roles {
		ids[i] = role.ID
	}
	return ids
}

func countRole(roles []domain.Role, roleID string) int {
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
