package application

import (
	"context"
	"minimart/internal/auth/domain"
	"minimart/internal/auth/infrastructure/memory"
	"minimart/pkg/metrics"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type assignRoleErrorRepo struct {
	memoryRepo *memory.Repository
	assignErr  error
}

func (r *assignRoleErrorRepo) Save(ctx context.Context, acc domain.Account) error {
	return r.memoryRepo.Save(ctx, acc)
}

func (r *assignRoleErrorRepo) GetByID(ctx context.Context, id string) (domain.Account, error) {
	return r.memoryRepo.GetByID(ctx, id)
}

func (r *assignRoleErrorRepo) GetByUsername(ctx context.Context, username string) (domain.Account, error) {
	return r.memoryRepo.GetByUsername(ctx, username)
}

func (r *assignRoleErrorRepo) UpdatePassword(ctx context.Context, acc domain.Account) error {
	return r.memoryRepo.UpdatePassword(ctx, acc)
}

func (r *assignRoleErrorRepo) GetRole(ctx context.Context, id string) (domain.Role, error) {
	return r.memoryRepo.GetRole(ctx, id)
}

func (r *assignRoleErrorRepo) GetPermission(ctx context.Context, id string) (domain.Permission, error) {
	return r.memoryRepo.GetPermission(ctx, id)
}

func (r *assignRoleErrorRepo) GetRolesForAccount(ctx context.Context, accountID string) ([]domain.Role, error) {
	return r.memoryRepo.GetRolesForAccount(ctx, accountID)
}

func (r *assignRoleErrorRepo) GetPermissionsForAccount(ctx context.Context, accountID string) ([]string, error) {
	return r.memoryRepo.GetPermissionsForAccount(ctx, accountID)
}

func (r *assignRoleErrorRepo) GetAccountsByUsername(ctx context.Context, usernames []string) ([]string, error) {
	return r.memoryRepo.GetAccountsByUsername(ctx, usernames)
}

func (r *assignRoleErrorRepo) AssignRole(ctx context.Context, accountID, roleID string) error {
	if r.assignErr != nil {
		return r.assignErr
	}
	return r.memoryRepo.AssignRole(ctx, accountID, roleID)
}

func (r *assignRoleErrorRepo) CreateRole(ctx context.Context, r2 domain.Role) error {
	return r.memoryRepo.CreateRole(ctx, r2)
}

func (r *assignRoleErrorRepo) CreatePermission(ctx context.Context, p domain.Permission) error {
	return r.memoryRepo.CreatePermission(ctx, p)
}

func (r *assignRoleErrorRepo) AddRolePermission(ctx context.Context, roleID, permissionID string) error {
	return r.memoryRepo.AddRolePermission(ctx, roleID, permissionID)
}

var _ domain.Repository = (*assignRoleErrorRepo)(nil)

func TestAssignRole(t *testing.T) {
	t.Run("assigns role successfully", func(t *testing.T) {
		svc, repo := newTestService(t, &fakeProvisioner{})

		acc, _ := svc.SignUp(t.Context(), SignUpRequest{Username: "testuser", Password: "password123", Email: "test@example.com"})

		// Seed the role first
		_ = repo.CreateRole(t.Context(), domain.Role{ID: "support", Description: "Support"})

		err := svc.AssignRole(t.Context(), acc.ID, "support")
		require.NoError(t, err)

		roles, err := repo.GetRolesForAccount(t.Context(), acc.ID)
		require.NoError(t, err)
		assert.Len(t, roles, 1)
		assert.Equal(t, "support", roles[0].ID)
	})

	t.Run("idempotent re-assign swallows ErrRoleAlreadyAssigned", func(t *testing.T) {
		svc, _ := newTestService(t, &fakeProvisioner{})

		acc, _ := svc.SignUp(t.Context(), SignUpRequest{Username: "testuser2", Password: "password123", Email: "test2@example.com"})

		err := svc.AssignRole(t.Context(), acc.ID, "support")
		require.NoError(t, err)

		err = svc.AssignRole(t.Context(), acc.ID, "support")
		require.NoError(t, err)
	})

	t.Run("propagates unknown role error", func(t *testing.T) {
		repo := memory.New()
		errRepo := &assignRoleErrorRepo{memoryRepo: repo, assignErr: domain.ErrRoleNotFound}
		svc := NewService(errRepo, nil, metrics.Noop{})

		acc, _ := svc.SignUp(t.Context(), SignUpRequest{Username: "testuser3", Password: "password123", Email: "test3@example.com"})

		err := svc.AssignRole(t.Context(), acc.ID, "nonexistent")
		require.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrRoleNotFound)
	})
}

func TestAssignDefaultRole(t *testing.T) {
	t.Run("assigns customer role", func(t *testing.T) {
		svc, repo := newTestService(t, &fakeProvisioner{})

		acc, _ := svc.SignUp(t.Context(), SignUpRequest{Username: "testuser", Password: "password123", Email: "test@example.com"})

		_ = repo.CreateRole(t.Context(), domain.Role{ID: "customer", Description: "Customer"})

		err := svc.AssignDefaultRole(t.Context(), acc.ID)
		require.NoError(t, err)

		roles, err := repo.GetRolesForAccount(t.Context(), acc.ID)
		require.NoError(t, err)
		assert.Len(t, roles, 1)
		assert.Equal(t, "customer", roles[0].ID)
	})
}
