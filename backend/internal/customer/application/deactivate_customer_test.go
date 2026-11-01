package application

import (
	"context"
	"errors"
	"minimart/internal/customer/domain"
	"minimart/internal/customer/infrastructure/memory"
	"minimart/pkg/metrics"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDeactivateCustomer(t *testing.T) {
	t.Run("deactivates an active customer", func(t *testing.T) {
		repo := memory.New()
		svc := NewService(repo, metrics.Noop{})

		created, err := svc.CreateCustomer(t.Context(), CreateCustomerRequest{
			UserID:      "user-1",
			DisplayName: "John",
		})
		require.NoError(t, err)
		assert.Equal(t, domain.StatusActive, created.Status)
		assert.Equal(t, 1, created.Version)

		deactivated, err := svc.DeactivateCustomer(t.Context(), created.ID)
		require.NoError(t, err)
		assert.Equal(t, domain.StatusDeactivated, deactivated.Status)
		assert.NotNil(t, deactivated.DeactivatedAt)
		assert.Equal(t, created.ID, deactivated.ID)
		assert.Equal(t, "user-1", deactivated.UserID)
		assert.Equal(t, created.Version+1, deactivated.Version)

		got, err := svc.GetCustomerByID(t.Context(), created.ID)
		require.NoError(t, err)
		assert.Equal(t, domain.StatusDeactivated, got.Status)
		assert.NotNil(t, got.DeactivatedAt)
		assert.Equal(t, created.Version+1, got.Version)
	})

	t.Run("idempotent: deactivating already deactivated customer returns it unchanged (status/deactivated_at), version bumps", func(t *testing.T) {
		repo := memory.New()
		svc := NewService(repo, metrics.Noop{})

		created, err := svc.CreateCustomer(t.Context(), CreateCustomerRequest{
			UserID: "user-1",
		})
		require.NoError(t, err)

		deact1, err := svc.DeactivateCustomer(t.Context(), created.ID)
		require.NoError(t, err)

		deact2, err := svc.DeactivateCustomer(t.Context(), created.ID)
		require.NoError(t, err)

		assert.Equal(t, deact1.ID, deact2.ID)
		assert.Equal(t, deact1.Status, deact2.Status)
		assert.Equal(t, deact1.DeactivatedAt, deact2.DeactivatedAt)
		assert.Equal(t, deact1.Version+1, deact2.Version)
	})

	t.Run("returns ErrCustomerNotFound for unknown ID", func(t *testing.T) {
		repo := memory.New()
		svc := NewService(repo, metrics.Noop{})

		_, err := svc.DeactivateCustomer(t.Context(), "unknown")
		require.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrCustomerNotFound)
	})
}

type deactivateErrRepo struct {
	getErr        error
	deactivateErr error
}

var _ domain.Repository = (*deactivateErrRepo)(nil)

func (r *deactivateErrRepo) GetByID(context.Context, string) (domain.Customer, error) {
	if r.getErr != nil {
		return domain.Customer{}, r.getErr
	}
	// Return a valid customer so the use case can proceed to Deactivate
	now := time.Now().UTC()
	return domain.Customer{
		ID:            "cust-1",
		UserID:        "user-1",
		DisplayName:   "John",
		Status:        domain.StatusActive,
		CreatedAt:     now,
		UpdatedAt:     now,
		DeactivatedAt: nil,
		Version:       1,
	}, nil
}

func (r *deactivateErrRepo) GetByUserID(context.Context, string) (domain.Customer, error) {
	return domain.Customer{}, domain.ErrCustomerNotFound
}

func (r *deactivateErrRepo) Save(context.Context, domain.Customer) error {
	return nil
}

func (r *deactivateErrRepo) UpdateProfile(context.Context, domain.Customer) (domain.Customer, error) {
	return domain.Customer{}, nil
}

func (r *deactivateErrRepo) Deactivate(context.Context, domain.Customer) (domain.Customer, error) {
	return domain.Customer{}, r.deactivateErr
}

func TestDeactivateCustomerErrors(t *testing.T) {
	t.Run("propagates unexpected lookup error", func(t *testing.T) {
		svc := NewService(&deactivateErrRepo{getErr: errors.New("boom")}, metrics.Noop{})
		_, err := svc.DeactivateCustomer(t.Context(), "id")
		require.Error(t, err)
		assert.ErrorContains(t, err, "boom")
	})

	t.Run("propagates deactivate error", func(t *testing.T) {
		svc := NewService(&deactivateErrRepo{deactivateErr: errors.New("db down")}, metrics.Noop{})
		_, err := svc.DeactivateCustomer(t.Context(), "id")
		require.Error(t, err)
		assert.ErrorContains(t, err, "db down")
	})
}
