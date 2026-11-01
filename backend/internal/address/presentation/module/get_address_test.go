package module

import (
	"context"
	"errors"
	"minimart/internal/address/application"
	"minimart/internal/address/domain"
	"minimart/internal/address/infrastructure/memory"
	"minimart/internal/address/presentation/module/client"
	"minimart/pkg/metrics"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type failingRepo struct {
	err error
}

func (f failingRepo) GetByID(context.Context, string) (domain.Address, error) {
	return domain.Address{}, f.err
}

func (f failingRepo) GetByCustomerID(context.Context, string) ([]domain.Address, error) {
	return nil, nil
}

func (f failingRepo) GetDefaultByCustomerID(context.Context, string) (domain.Address, error) {
	return domain.Address{}, f.err
}
func (f failingRepo) Save(context.Context, domain.Address) error { return nil }
func (f failingRepo) Update(context.Context, domain.Address) (domain.Address, error) {
	return domain.Address{}, nil
}
func (f failingRepo) Delete(context.Context, string, string) error { return nil }
func (f failingRepo) SetDefault(context.Context, string, string) (domain.Address, error) {
	return domain.Address{}, nil
}

func TestGetAddress(t *testing.T) {
	addr, err := domain.NewAddress("addr-1", "cust-1", domain.AddressFields{Label: "Home", FullName: "John Doe", Street: "123 Main St", City: "New York", State: "NY", Zip: "10001", Country: "USA"}, true)
	require.NoError(t, err)

	t.Run("maps a domain address into the client type", func(t *testing.T) {
		repo := memory.New()
		require.NoError(t, repo.Save(t.Context(), addr))
		adapter := New(application.NewService(repo, metrics.Noop{}))

		got, err := adapter.GetAddress(t.Context(), client.GetAddressRequest{ID: "addr-1", CustomerID: "cust-1"})
		require.NoError(t, err)
		assert.Equal(t, "addr-1", got.ID)
		assert.Equal(t, "cust-1", got.CustomerID)
		assert.Equal(t, "Home", got.Label)
		assert.Equal(t, "John Doe", got.FullName)
	})

	t.Run("translates not-found into the client sentinel", func(t *testing.T) {
		adapter := New(application.NewService(memory.New(), metrics.Noop{}))

		_, err := adapter.GetAddress(t.Context(), client.GetAddressRequest{ID: "missing", CustomerID: "cust-1"})
		assert.ErrorIs(t, err, client.ErrAddressNotFound)
	})

	t.Run("passes through unexpected errors unchanged", func(t *testing.T) {
		boom := errors.New("boom")
		adapter := New(application.NewService(failingRepo{err: boom}, metrics.Noop{}))

		_, err := adapter.GetAddress(t.Context(), client.GetAddressRequest{ID: "addr-1", CustomerID: "cust-1"})
		require.ErrorIs(t, err, boom)
		assert.NotErrorIs(t, err, client.ErrAddressNotFound)
	})
}
