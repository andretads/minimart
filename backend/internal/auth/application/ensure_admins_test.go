package application

import (
	"context"
	"errors"
	"minimart/internal/auth/domain"
	"minimart/internal/auth/infrastructure/memory"
	"minimart/pkg/metrics"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type adminErrorRepo struct {
	memoryRepo *memory.Repository
	getErr     error
}

func (r *adminErrorRepo) Save(ctx context.Context, acc domain.Account) error {
	return r.memoryRepo.Save(ctx, acc)
}

func (r *adminErrorRepo) GetByID(ctx context.Context, id string) (domain.Account, error) {
	return r.memoryRepo.GetByID(ctx, id)
}

func (r *adminErrorRepo) GetByUsername(ctx context.Context, username string) (domain.Account, error) {
	return r.memoryRepo.GetByUsername(ctx, username)
}

func (r *adminErrorRepo) UpdatePassword(ctx context.Context, acc domain.Account) error {
	return r.memoryRepo.UpdatePassword(ctx, acc)
}

func (r *adminErrorRepo) GetRole(ctx context.Context, id string) (domain.Role, error) {
	return r.memoryRepo.GetRole(ctx, id)
}

func (r *adminErrorRepo) GetPermission(ctx context.Context, id string) (domain.Permission, error) {
	return r.memoryRepo.GetPermission(ctx, id)
}

func (r *adminErrorRepo) GetRolesForAccount(ctx context.Context, accountID string) ([]domain.Role, error) {
	return r.memoryRepo.GetRolesForAccount(ctx, accountID)
}

func (r *adminErrorRepo) GetPermissionsForAccount(ctx context.Context, accountID string) ([]string, error) {
	return r.memoryRepo.GetPermissionsForAccount(ctx, accountID)
}

func (r *adminErrorRepo) GetAccountsByUsername(ctx context.Context, usernames []string) ([]string, error) {
	if r.getErr != nil {
		return nil, r.getErr
	}
	return r.memoryRepo.GetAccountsByUsername(ctx, usernames)
}

func (r *adminErrorRepo) AssignRole(ctx context.Context, accountID, roleID string) error {
	return r.memoryRepo.AssignRole(ctx, accountID, roleID)
}

func (r *adminErrorRepo) CreateRole(ctx context.Context, r2 domain.Role) error {
	return r.memoryRepo.CreateRole(ctx, r2)
}

func (r *adminErrorRepo) CreatePermission(ctx context.Context, p domain.Permission) error {
	return r.memoryRepo.CreatePermission(ctx, p)
}

func (r *adminErrorRepo) AddRolePermission(ctx context.Context, roleID, permissionID string) error {
	return r.memoryRepo.AddRolePermission(ctx, roleID, permissionID)
}

var _ domain.Repository = (*adminErrorRepo)(nil)

func TestEnsureAdmins(t *testing.T) {
	t.Run("empty usernames is no-op", func(t *testing.T) {
		svc, _ := newTestService(t, &fakeProvisioner{})

		err := svc.EnsureAdmins(t.Context(), []string{})
		require.NoError(t, err)
	})

	t.Run("assigns admin role to resolved usernames", func(t *testing.T) {
		svc, repo := newTestService(t, &fakeProvisioner{})

		acc1, _ := svc.SignUp(t.Context(), SignUpRequest{Username: "admin1", Password: "password123", Email: "admin1@example.com"})
		acc2, _ := svc.SignUp(t.Context(), SignUpRequest{Username: "admin2", Password: "password123", Email: "admin2@example.com"})

		// Seed the admin role
		_ = repo.CreateRole(t.Context(), domain.Role{ID: "admin", Description: "Administrator"})

		err := svc.EnsureAdmins(t.Context(), []string{"admin1", "admin2"})
		require.NoError(t, err)

		roles1, _ := repo.GetRolesForAccount(t.Context(), acc1.ID)
		roles2, _ := repo.GetRolesForAccount(t.Context(), acc2.ID)
		assert.Len(t, roles1, 1)
		assert.Equal(t, "admin", roles1[0].ID)
		assert.Len(t, roles2, 1)
		assert.Equal(t, "admin", roles2[0].ID)
	})

	t.Run("idempotent for existing admin role", func(t *testing.T) {
		svc, repo := newTestService(t, &fakeProvisioner{})

		acc, _ := svc.SignUp(t.Context(), SignUpRequest{Username: "admin1", Password: "password123", Email: "admin1@example.com"})
		_ = repo.CreateRole(t.Context(), domain.Role{ID: "admin", Description: "Administrator"})
		_ = repo.AssignRole(t.Context(), acc.ID, "admin")

		err := svc.EnsureAdmins(t.Context(), []string{"admin1"})
		require.NoError(t, err)

		roles, _ := repo.GetRolesForAccount(t.Context(), acc.ID)
		assert.Len(t, roles, 1)
		assert.Equal(t, "admin", roles[0].ID)
	})

	t.Run("propagates repo error", func(t *testing.T) {
		repo := memory.New()
		errRepo := &adminErrorRepo{memoryRepo: repo, getErr: errors.New("db error")}
		svc := NewService(errRepo, nil, metrics.Noop{})

		err := svc.EnsureAdmins(t.Context(), []string{"admin1"})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "db error")
	})
}
