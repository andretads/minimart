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

func TestCreateCustomer(t *testing.T) {
	t.Run("creates a customer with user_id and display_name", func(t *testing.T) {
		repo := memory.New()
		svc := NewService(repo, metrics.Noop{})

		got, err := svc.CreateCustomer(t.Context(), CreateCustomerRequest{
			UserID:      "user-1",
			DisplayName: "John",
		})
		require.NoError(t, err)
		assert.NotEmpty(t, got.ID)
		assert.Equal(t, "user-1", got.UserID)
		assert.Equal(t, "John", got.DisplayName)
		assert.Equal(t, domain.StatusActive, got.Status)
		assert.False(t, got.CreatedAt.IsZero())
		assert.False(t, got.UpdatedAt.IsZero())
		assert.Nil(t, got.DeactivatedAt)
		assert.Equal(t, 1, got.Version)
	})

	t.Run("creates a customer without display_name", func(t *testing.T) {
		repo := memory.New()
		svc := NewService(repo, metrics.Noop{})

		got, err := svc.CreateCustomer(t.Context(), CreateCustomerRequest{
			UserID: "user-2",
		})
		require.NoError(t, err)
		assert.NotEmpty(t, got.ID)
		assert.Equal(t, "user-2", got.UserID)
		assert.Empty(t, got.DisplayName)
		assert.Equal(t, 1, got.Version)
	})

	t.Run("rejects empty user_id", func(t *testing.T) {
		repo := memory.New()
		svc := NewService(repo, metrics.Noop{})

		_, err := svc.CreateCustomer(t.Context(), CreateCustomerRequest{
			UserID: "",
		})
		require.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrEmptyUserID)
	})

	t.Run("rejects duplicate user_id", func(t *testing.T) {
		repo := memory.New()
		svc := NewService(repo, metrics.Noop{})

		_, err := svc.CreateCustomer(t.Context(), CreateCustomerRequest{
			UserID: "user-1",
		})
		require.NoError(t, err)

		_, err = svc.CreateCustomer(t.Context(), CreateCustomerRequest{
			UserID: "user-1",
		})
		require.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrCustomerExists)
	})

	t.Run("trims display_name whitespace", func(t *testing.T) {
		repo := memory.New()
		svc := NewService(repo, metrics.Noop{})

		got, err := svc.CreateCustomer(t.Context(), CreateCustomerRequest{
			UserID:      "user-3",
			DisplayName: "  Jane  ",
		})
		require.NoError(t, err)
		assert.Equal(t, "Jane", got.DisplayName)
		assert.Equal(t, 1, got.Version)
	})
}

type createErrRepo struct {
	createErr error
}

var _ domain.Repository = (*createErrRepo)(nil)

func (r *createErrRepo) GetByID(context.Context, string) (domain.Customer, error) {
	return domain.Customer{}, domain.ErrCustomerNotFound
}

func (r *createErrRepo) GetByUserID(context.Context, string) (domain.Customer, error) {
	return domain.Customer{}, domain.ErrCustomerNotFound
}

func (r *createErrRepo) Save(context.Context, domain.Customer) error {
	return r.createErr
}

func (r *createErrRepo) UpdateProfile(context.Context, domain.Customer) (domain.Customer, error) {
	return domain.Customer{}, nil
}

func (r *createErrRepo) Deactivate(context.Context, domain.Customer) (domain.Customer, error) {
	return domain.Customer{}, nil
}

func TestCreateCustomerErrors(t *testing.T) {
	t.Run("propagates create error", func(t *testing.T) {
		svc := NewService(&createErrRepo{createErr: errors.New("db down")}, metrics.Noop{})
		_, err := svc.CreateCustomer(t.Context(), CreateCustomerRequest{UserID: "user-1"})
		require.Error(t, err)
		assert.ErrorContains(t, err, "db down")
	})
}
