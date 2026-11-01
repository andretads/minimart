package module

import (
	"context"
	"errors"
	"minimart/internal/customer/application"
	"minimart/internal/customer/domain"
	"minimart/internal/customer/infrastructure/memory"
	"minimart/pkg/metrics"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// failingRepo makes GetByUserID fail with a non-not-found error so the
// pass-through branch can be exercised.
type failingRepo struct {
	err error
}

var _ domain.Repository = (*failingRepo)(nil)

func (f *failingRepo) GetByID(context.Context, string) (domain.Customer, error) {
	return domain.Customer{}, f.err
}

func (f *failingRepo) GetByUserID(context.Context, string) (domain.Customer, error) {
	return domain.Customer{}, f.err
}

func (f *failingRepo) Save(context.Context, domain.Customer) error { return nil }

func (f *failingRepo) UpdateProfile(context.Context, domain.Customer) (domain.Customer, error) {
	return domain.Customer{}, nil
}

func (f *failingRepo) Deactivate(context.Context, domain.Customer) (domain.Customer, error) {
	return domain.Customer{}, nil
}

func TestAdapterEnsureCustomer(t *testing.T) {
	t.Run("maps a domain customer into the client type", func(t *testing.T) {
		repo := memory.New()
		adapter := New(application.NewService(repo, metrics.Noop{}))

		got, err := adapter.EnsureCustomer(t.Context(), "user-1")
		require.NoError(t, err)
		assert.NotEmpty(t, got.ID)
		assert.Equal(t, "user-1", got.UserID)
		assert.Empty(t, got.DisplayName)
		assert.Equal(t, "active", got.Status)

		stored, err := repo.GetByUserID(t.Context(), "user-1")
		require.NoError(t, err)
		assert.Equal(t, stored.ID, got.ID)
	})

	t.Run("passes through unexpected errors unchanged", func(t *testing.T) {
		boom := errors.New("boom")
		adapter := New(application.NewService(&failingRepo{err: boom}, metrics.Noop{}))

		_, err := adapter.EnsureCustomer(t.Context(), "user-1")
		assert.ErrorIs(t, err, boom)
	})
}
