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

type permErrorRepo struct {
	memoryRepo *memory.Repository
	getErr     error
}

func (r *permErrorRepo) Save(ctx context.Context, acc domain.Account) error {
	return r.memoryRepo.Save(ctx, acc)
}

func (r *permErrorRepo) GetByID(ctx context.Context, id string) (domain.Account, error) {
	return r.memoryRepo.GetByID(ctx, id)
}

func (r *permErrorRepo) GetByUsername(ctx context.Context, username string) (domain.Account, error) {
	return r.memoryRepo.GetByUsername(ctx, username)
}

func (r *permErrorRepo) UpdatePassword(ctx context.Context, acc domain.Account) error {
	return r.memoryRepo.UpdatePassword(ctx, acc)
}

func (r *permErrorRepo) GetRole(ctx context.Context, id string) (domain.Role, error) {
	return r.memoryRepo.GetRole(ctx, id)
}

func (r *permErrorRepo) GetPermission(ctx context.Context, id string) (domain.Permission, error) {
	return r.memoryRepo.GetPermission(ctx, id)
}

func (r *permErrorRepo) GetRolesForAccount(ctx context.Context, accountID string) ([]domain.Role, error) {
	return r.memoryRepo.GetRolesForAccount(ctx, accountID)
}

func (r *permErrorRepo) GetPermissionsForAccount(ctx context.Context, accountID string) ([]string, error) {
	if r.getErr != nil {
		return nil, r.getErr
	}
	return r.memoryRepo.GetPermissionsForAccount(ctx, accountID)
}

func (r *permErrorRepo) GetAccountsByUsername(ctx context.Context, usernames []string) ([]string, error) {
	return r.memoryRepo.GetAccountsByUsername(ctx, usernames)
}

func (r *permErrorRepo) AssignRole(ctx context.Context, accountID, roleID string) error {
	return r.memoryRepo.AssignRole(ctx, accountID, roleID)
}

func (r *permErrorRepo) CreateRole(ctx context.Context, r2 domain.Role) error {
	return r.memoryRepo.CreateRole(ctx, r2)
}

func (r *permErrorRepo) CreatePermission(ctx context.Context, p domain.Permission) error {
	return r.memoryRepo.CreatePermission(ctx, p)
}

func (r *permErrorRepo) AddRolePermission(ctx context.Context, roleID, permissionID string) error {
	return r.memoryRepo.AddRolePermission(ctx, roleID, permissionID)
}

var _ domain.Repository = (*permErrorRepo)(nil)

func TestPermissionsForAccount(t *testing.T) {
	t.Run("returns permissions for account", func(t *testing.T) {
		svc, repo := newTestService(t, &fakeProvisioner{})

		acc, _ := svc.SignUp(t.Context(), SignUpRequest{Username: "testuser", Password: "password123", Email: "test@example.com"})
		_ = repo.CreateRole(t.Context(), domain.Role{ID: "customer", Description: "Customer"})
		_ = repo.CreatePermission(t.Context(), domain.Permission{ID: "catalog:read"})
		_ = repo.AddRolePermission(t.Context(), "customer", "catalog:read")
		_ = repo.AssignRole(t.Context(), acc.ID, "customer")

		perms, err := svc.PermissionsForAccount(t.Context(), acc.ID)
		require.NoError(t, err)
		assert.Contains(t, perms, "catalog:read")
	})

	t.Run("returns error from repo", func(t *testing.T) {
		repo := memory.New()
		errRepo := &permErrorRepo{memoryRepo: repo, getErr: errors.New("db error")}
		svc := NewService(errRepo, nil, metrics.Noop{})

		_, err := svc.PermissionsForAccount(t.Context(), "some-id")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "db error")
	})
}
