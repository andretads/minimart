package application

import (
	"context"
	"errors"
	"minimart/internal/customer/domain"
	"minimart/internal/customer/infrastructure/memory"
	"minimart/pkg/metrics"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEnsureCustomer(t *testing.T) {
	t.Run("creates a customer for a new user id", func(t *testing.T) {
		repo := memory.New()
		svc := NewService(repo, metrics.Noop{})

		c, err := svc.EnsureCustomer(t.Context(), "user-1")
		require.NoError(t, err)
		assert.NotEmpty(t, c.ID)
		assert.Equal(t, "user-1", c.UserID)

		got, err := repo.GetByUserID(t.Context(), "user-1")
		require.NoError(t, err)
		assert.Equal(t, c.ID, got.ID)
	})

	t.Run("returns the existing customer for a known user id", func(t *testing.T) {
		repo := memory.New()
		svc := NewService(repo, metrics.Noop{})
		c1, err := svc.EnsureCustomer(t.Context(), "user-1")
		require.NoError(t, err)

		c2, err := svc.EnsureCustomer(t.Context(), "user-1")
		require.NoError(t, err)
		assert.Equal(t, c1.ID, c2.ID)
	})

	t.Run("rejects an empty user id", func(t *testing.T) {
		repo := memory.New()
		svc := NewService(repo, metrics.Noop{})

		_, err := svc.EnsureCustomer(t.Context(), "")
		require.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrEmptyUserID)
	})
}

type errRepo struct {
	getErr    error
	createErr error
}

var _ domain.Repository = (*errRepo)(nil)

func (r *errRepo) GetByID(context.Context, string) (domain.Customer, error) {
	return domain.Customer{}, domain.ErrCustomerNotFound
}

func (r *errRepo) GetByUserID(context.Context, string) (domain.Customer, error) {
	if r.getErr != nil {
		return domain.Customer{}, r.getErr
	}
	return domain.Customer{}, domain.ErrCustomerNotFound
}

func (r *errRepo) Save(context.Context, domain.Customer) error {
	return r.createErr
}

func (r *errRepo) UpdateProfile(context.Context, domain.Customer) (domain.Customer, error) {
	return domain.Customer{}, nil
}

func (r *errRepo) Deactivate(context.Context, domain.Customer) (domain.Customer, error) {
	return domain.Customer{}, nil
}

func TestEnsureCustomerErrors(t *testing.T) {
	t.Run("propagates unexpected lookup error", func(t *testing.T) {
		svc := NewService(&errRepo{getErr: errors.New("boom")}, metrics.Noop{})
		_, err := svc.EnsureCustomer(t.Context(), "user-1")
		require.Error(t, err)
		assert.ErrorContains(t, err, "boom")
	})

	t.Run("propagates create error", func(t *testing.T) {
		svc := NewService(&errRepo{createErr: errors.New("db down")}, metrics.Noop{})
		_, err := svc.EnsureCustomer(t.Context(), "user-1")
		require.Error(t, err)
		assert.ErrorContains(t, err, "db down")
	})
}
