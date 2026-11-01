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

func TestGetCustomerByUserID(t *testing.T) {
	t.Run("returns customer by user_id", func(t *testing.T) {
		repo := memory.New()
		svc := NewService(repo, metrics.Noop{})

		created, err := svc.CreateCustomer(t.Context(), CreateCustomerRequest{
			UserID:      "user-1",
			DisplayName: "John",
		})
		require.NoError(t, err)

		got, err := svc.GetCustomerByUserID(t.Context(), "user-1")
		require.NoError(t, err)
		assert.Equal(t, created.ID, got.ID)
		assert.Equal(t, "user-1", got.UserID)
		assert.Equal(t, "John", got.DisplayName)
		assert.Equal(t, domain.StatusActive, got.Status)
	})

	t.Run("returns ErrCustomerNotFound for unknown user_id", func(t *testing.T) {
		repo := memory.New()
		svc := NewService(repo, metrics.Noop{})

		_, err := svc.GetCustomerByUserID(t.Context(), "unknown")
		require.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrCustomerNotFound)
	})
}

type errRepoGetByUserID struct {
	getErr error
}

var _ domain.Repository = (*errRepoGetByUserID)(nil)

func (r *errRepoGetByUserID) GetByID(context.Context, string) (domain.Customer, error) {
	return domain.Customer{}, domain.ErrCustomerNotFound
}

func (r *errRepoGetByUserID) GetByUserID(context.Context, string) (domain.Customer, error) {
	if r.getErr != nil {
		return domain.Customer{}, r.getErr
	}
	return domain.Customer{}, domain.ErrCustomerNotFound
}

func (r *errRepoGetByUserID) Save(context.Context, domain.Customer) error {
	return nil
}

func (r *errRepoGetByUserID) UpdateProfile(context.Context, domain.Customer) (domain.Customer, error) {
	return domain.Customer{}, nil
}

func (r *errRepoGetByUserID) Deactivate(context.Context, domain.Customer) (domain.Customer, error) {
	return domain.Customer{}, nil
}

func TestGetCustomerByUserIDErrors(t *testing.T) {
	t.Run("propagates unexpected error", func(t *testing.T) {
		svc := NewService(&errRepoGetByUserID{getErr: errors.New("boom")}, metrics.Noop{})
		_, err := svc.GetCustomerByUserID(t.Context(), "user-1")
		require.Error(t, err)
		assert.ErrorContains(t, err, "boom")
	})
}
