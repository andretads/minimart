package module

import (
	"context"
	"errors"
	"minimart/internal/customer/application"
	"minimart/internal/customer/domain"
	"minimart/internal/customer/infrastructure/memory"
	"minimart/internal/customer/presentation/module/client"
	"minimart/pkg/metrics"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// failingRepoForGetByUserID makes GetByUserID fail with a non-not-found error so the
// pass-through branch can be exercised.
type failingRepoForGetByUserID struct {
	err error
}

var _ domain.Repository = (*failingRepoForGetByUserID)(nil)

func (f *failingRepoForGetByUserID) GetByID(context.Context, string) (domain.Customer, error) {
	return domain.Customer{}, domain.ErrCustomerNotFound
}

func (f *failingRepoForGetByUserID) GetByUserID(context.Context, string) (domain.Customer, error) {
	return domain.Customer{}, f.err
}

func (f *failingRepoForGetByUserID) Save(context.Context, domain.Customer) error { return nil }

func (f *failingRepoForGetByUserID) UpdateProfile(context.Context, domain.Customer) (domain.Customer, error) {
	return domain.Customer{}, nil
}

func (f *failingRepoForGetByUserID) Deactivate(context.Context, domain.Customer) (domain.Customer, error) {
	return domain.Customer{}, nil
}

func TestAdapterGetCustomerByUserID(t *testing.T) {
	t.Run("returns the customer from the service", func(t *testing.T) {
		repo := memory.New()
		svc := application.NewService(repo, metrics.Noop{})
		created, err := svc.CreateCustomer(t.Context(), application.CreateCustomerRequest{
			UserID: "user-1", DisplayName: "John",
		})
		require.NoError(t, err)

		adapter := New(svc)
		got, err := adapter.GetCustomerByUserID(t.Context(), "user-1")
		require.NoError(t, err)
		assert.Equal(t, created.ID, got.ID)
		assert.Equal(t, "user-1", got.UserID)
		assert.Equal(t, "John", got.DisplayName)
		assert.Equal(t, "active", got.Status)
	})

	t.Run("passes through not found error", func(t *testing.T) {
		adapter := New(application.NewService(&failingRepoForGetByUserID{err: domain.ErrCustomerNotFound}, metrics.Noop{}))
		_, err := adapter.GetCustomerByUserID(t.Context(), "missing")
		assert.ErrorIs(t, err, client.ErrCustomerNotFound)
	})

	t.Run("passes through unexpected errors unchanged", func(t *testing.T) {
		boom := errors.New("boom")
		adapter := New(application.NewService(&failingRepoForGetByUserID{err: boom}, metrics.Noop{}))

		_, err := adapter.GetCustomerByUserID(t.Context(), "missing")
		require.ErrorIs(t, err, boom)
		assert.NotErrorIs(t, err, client.ErrCustomerNotFound)
	})
}
