package application

import (
	"context"
	"errors"
	"minimart/internal/customer/domain"
	"minimart/internal/customer/infrastructure/memory"
	"minimart/pkg/metrics"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUpdateCustomerProfile(t *testing.T) {
	t.Run("updates display_name", func(t *testing.T) {
		repo := memory.New()
		svc := NewService(repo, metrics.Noop{})

		created, err := svc.CreateCustomer(t.Context(), CreateCustomerRequest{
			UserID:      "user-1",
			DisplayName: "John",
		})
		require.NoError(t, err)

		updated, err := svc.UpdateCustomerProfile(t.Context(), UpdateCustomerProfileRequest{
			ID:          created.ID,
			DisplayName: "Jane",
			Version:     created.Version,
		})
		require.NoError(t, err)
		assert.Equal(t, "Jane", updated.DisplayName)
		assert.Equal(t, created.ID, updated.ID)
		assert.Equal(t, "user-1", updated.UserID)
		assert.Equal(t, created.Version+1, updated.Version)

		got, err := svc.GetCustomerByID(t.Context(), created.ID)
		require.NoError(t, err)
		assert.Equal(t, "Jane", got.DisplayName)
	})

	t.Run("clears display_name when empty string provided", func(t *testing.T) {
		repo := memory.New()
		svc := NewService(repo, metrics.Noop{})

		created, err := svc.CreateCustomer(t.Context(), CreateCustomerRequest{
			UserID:      "user-1",
			DisplayName: "John",
		})
		require.NoError(t, err)

		updated, err := svc.UpdateCustomerProfile(t.Context(), UpdateCustomerProfileRequest{
			ID:          created.ID,
			DisplayName: "",
			Version:     created.Version,
		})
		require.NoError(t, err)
		assert.Empty(t, updated.DisplayName)
	})

	t.Run("trims display_name whitespace", func(t *testing.T) {
		repo := memory.New()
		svc := NewService(repo, metrics.Noop{})

		created, err := svc.CreateCustomer(t.Context(), CreateCustomerRequest{
			UserID: "user-1",
		})
		require.NoError(t, err)

		updated, err := svc.UpdateCustomerProfile(t.Context(), UpdateCustomerProfileRequest{
			ID:          created.ID,
			DisplayName: "  Jane  ",
			Version:     created.Version,
		})
		require.NoError(t, err)
		assert.Equal(t, "Jane", updated.DisplayName)
	})

	t.Run("rejects display_name too long", func(t *testing.T) {
		repo := memory.New()
		svc := NewService(repo, metrics.Noop{})

		created, err := svc.CreateCustomer(t.Context(), CreateCustomerRequest{
			UserID: "user-1",
		})
		require.NoError(t, err)

		longName := strings.Repeat("a", 201)

		_, err = svc.UpdateCustomerProfile(t.Context(), UpdateCustomerProfileRequest{
			ID:          created.ID,
			DisplayName: longName,
			Version:     created.Version,
		})
		require.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrDisplayNameTooLong)
	})

	t.Run("returns ErrCustomerNotFound for unknown ID", func(t *testing.T) {
		repo := memory.New()
		svc := NewService(repo, metrics.Noop{})

		_, err := svc.UpdateCustomerProfile(t.Context(), UpdateCustomerProfileRequest{
			ID:          "unknown",
			DisplayName: "Jane",
			Version:     1,
		})
		require.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrCustomerNotFound)
	})

	t.Run("rejects missing version", func(t *testing.T) {
		repo := memory.New()
		svc := NewService(repo, metrics.Noop{})

		created, err := svc.CreateCustomer(t.Context(), CreateCustomerRequest{
			UserID: "user-1",
		})
		require.NoError(t, err)

		_, err = svc.UpdateCustomerProfile(t.Context(), UpdateCustomerProfileRequest{
			ID:          created.ID,
			DisplayName: "Jane",
			Version:     0,
		})
		require.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrInvalidVersion)
	})

	t.Run("rejects negative version", func(t *testing.T) {
		repo := memory.New()
		svc := NewService(repo, metrics.Noop{})

		created, err := svc.CreateCustomer(t.Context(), CreateCustomerRequest{
			UserID: "user-1",
		})
		require.NoError(t, err)

		_, err = svc.UpdateCustomerProfile(t.Context(), UpdateCustomerProfileRequest{
			ID:          created.ID,
			DisplayName: "Jane",
			Version:     -1,
		})
		require.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrInvalidVersion)
	})

	t.Run("rejects stale version (optimistic lock)", func(t *testing.T) {
		repo := memory.New()
		svc := NewService(repo, metrics.Noop{})

		created, err := svc.CreateCustomer(t.Context(), CreateCustomerRequest{
			UserID:      "user-1",
			DisplayName: "John",
		})
		require.NoError(t, err)

		// First update succeeds
		_, err = svc.UpdateCustomerProfile(t.Context(), UpdateCustomerProfileRequest{
			ID:          created.ID,
			DisplayName: "Jane",
			Version:     created.Version,
		})
		require.NoError(t, err)

		// Second update with stale version fails
		_, err = svc.UpdateCustomerProfile(t.Context(), UpdateCustomerProfileRequest{
			ID:          created.ID,
			DisplayName: "Jane2",
			Version:     created.Version, // stale!
		})
		require.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrCustomerOptimisticLock)
	})
}

type updateErrRepo struct {
	getErr    error
	updateErr error
}

var _ domain.Repository = (*updateErrRepo)(nil)

func (r *updateErrRepo) GetByID(context.Context, string) (domain.Customer, error) {
	if r.getErr != nil {
		return domain.Customer{}, r.getErr
	}
	// Return a valid customer so the use case can proceed to UpdateProfile
	now := time.Now().UTC()
	return domain.Customer{
		ID:          "cust-1",
		UserID:      "user-1",
		DisplayName: "John",
		Status:      domain.StatusActive,
		CreatedAt:   now,
		UpdatedAt:   now,
		Version:     1,
	}, nil
}

func (r *updateErrRepo) GetByUserID(context.Context, string) (domain.Customer, error) {
	return domain.Customer{}, domain.ErrCustomerNotFound
}

func (r *updateErrRepo) Save(context.Context, domain.Customer) error {
	return nil
}

func (r *updateErrRepo) UpdateProfile(context.Context, domain.Customer) (domain.Customer, error) {
	return domain.Customer{}, r.updateErr
}

func (r *updateErrRepo) Deactivate(context.Context, domain.Customer) (domain.Customer, error) {
	return domain.Customer{}, nil
}

func TestUpdateCustomerProfileErrors(t *testing.T) {
	t.Run("propagates unexpected lookup error", func(t *testing.T) {
		svc := NewService(&updateErrRepo{getErr: errors.New("boom")}, metrics.Noop{})
		_, err := svc.UpdateCustomerProfile(t.Context(), UpdateCustomerProfileRequest{
			ID: "id", DisplayName: "Jane", Version: 1,
		})
		require.Error(t, err)
		assert.ErrorContains(t, err, "boom")
	})

	t.Run("propagates update error", func(t *testing.T) {
		svc := NewService(&updateErrRepo{updateErr: errors.New("db down")}, metrics.Noop{})
		_, err := svc.UpdateCustomerProfile(t.Context(), UpdateCustomerProfileRequest{
			ID: "id", DisplayName: "Jane", Version: 1,
		})
		require.Error(t, err)
		assert.ErrorContains(t, err, "db down")
	})
}
