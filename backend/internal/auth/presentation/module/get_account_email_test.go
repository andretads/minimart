package module

import (
	"context"
	"errors"
	"minimart/internal/auth/application"
	"minimart/internal/auth/domain"
	"minimart/internal/auth/infrastructure/memory"
	"minimart/internal/auth/presentation/module/client"
	customerclient "minimart/internal/customer/presentation/module/client"
	"minimart/pkg/metrics"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeCustomerProvisioner struct {
	err error
}

func (f fakeCustomerProvisioner) EnsureCustomer(ctx context.Context, userID string) (customerclient.Customer, error) {
	if f.err != nil {
		return customerclient.Customer{}, f.err
	}
	return customerclient.Customer{ID: "cust-1", UserID: userID}, nil
}

type failingRepo struct {
	err error
}

func (f failingRepo) GetByID(context.Context, string) (domain.Account, error) {
	return domain.Account{}, f.err
}

func (f failingRepo) GetByUsername(context.Context, string) (domain.Account, error) {
	return domain.Account{}, domain.ErrAccountNotFound
}

func (f failingRepo) Save(context.Context, domain.Account) error { return nil }

func (f failingRepo) UpdatePassword(context.Context, domain.Account) error { return nil }

func (f failingRepo) GetRole(context.Context, string) (domain.Role, error) {
	return domain.Role{}, domain.ErrRoleNotFound
}

func (f failingRepo) GetPermission(context.Context, string) (domain.Permission, error) {
	return domain.Permission{}, domain.ErrPermissionNotFound
}

func (f failingRepo) GetRolesForAccount(context.Context, string) ([]domain.Role, error) {
	return nil, nil
}

func (f failingRepo) GetPermissionsForAccount(context.Context, string) ([]string, error) {
	return nil, nil
}

func (f failingRepo) GetAccountsByUsername(context.Context, []string) ([]string, error) {
	return nil, nil
}
func (f failingRepo) AssignRole(context.Context, string, string) error          { return nil }
func (f failingRepo) CreateRole(context.Context, domain.Role) error             { return nil }
func (f failingRepo) CreatePermission(context.Context, domain.Permission) error { return nil }
func (f failingRepo) AddRolePermission(context.Context, string, string) error   { return nil }

var now = time.Now().UTC()

func TestAccountReaderAdapterGetAccountEmail(t *testing.T) {
	acc := domain.LoadAccount("acc-1", "testuser", "test@example.com", []byte("hash"), now, now)
	repo := memory.New()
	require.NoError(t, repo.Save(t.Context(), acc))

	t.Run("returns email for existing account", func(t *testing.T) {
		adapter := New(application.NewService(repo, fakeCustomerProvisioner{}, metrics.Noop{}))

		got, err := adapter.GetAccountEmail(t.Context(), "acc-1")
		require.NoError(t, err)
		assert.Equal(t, "test@example.com", got)
	})

	t.Run("translates not-found into client sentinel", func(t *testing.T) {
		adapter := New(application.NewService(memory.New(), fakeCustomerProvisioner{}, metrics.Noop{}))

		_, err := adapter.GetAccountEmail(t.Context(), "missing")
		assert.ErrorIs(t, err, client.ErrAccountNotFound)
	})

	t.Run("translates empty email into client sentinel", func(t *testing.T) {
		accNoEmail := domain.LoadAccount("acc-2", "testuser2", "", []byte("hash"), now, now)
		repo := memory.New()
		require.NoError(t, repo.Save(t.Context(), accNoEmail))
		adapter := New(application.NewService(repo, fakeCustomerProvisioner{}, metrics.Noop{}))

		_, err := adapter.GetAccountEmail(t.Context(), "acc-2")
		assert.ErrorIs(t, err, client.ErrEmailNotFound)
	})

	t.Run("passes through unexpected errors unchanged", func(t *testing.T) {
		boom := errors.New("boom")
		adapter := New(application.NewService(failingRepo{err: boom}, fakeCustomerProvisioner{}, metrics.Noop{}))

		_, err := adapter.GetAccountEmail(t.Context(), "acc-1")
		require.ErrorIs(t, err, boom)
		require.NotErrorIs(t, err, client.ErrAccountNotFound)
		require.NotErrorIs(t, err, client.ErrEmailNotFound)
	})
}
