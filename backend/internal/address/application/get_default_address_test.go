package application

import (
	"minimart/internal/address/domain"
	"minimart/internal/address/infrastructure/memory"
	"minimart/pkg/metrics"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetDefaultAddress(t *testing.T) {
	t.Run("returns the default address", func(t *testing.T) {
		repo := memory.New()
		svc := NewService(repo, metrics.Noop{})

		created, err := svc.CreateAddress(t.Context(), CreateAddressRequest{
			CustomerID: "cust-1",
			Label:      "Home",
			FullName:   "John Doe",
			Street:     "123 Main St",
			City:       "New York",
			State:      "NY",
			Zip:        "10001",
			Country:    "USA",
		})
		require.NoError(t, err)

		_, err = svc.SetDefaultAddress(t.Context(), SetDefaultAddressRequest{
			ID:         created.ID,
			CustomerID: "cust-1",
		})
		require.NoError(t, err)

		got, err := svc.GetDefaultAddress(t.Context(), "cust-1")
		require.NoError(t, err)
		assert.Equal(t, created.ID, got.ID)
		assert.Equal(t, "Home", got.Label)
	})

	t.Run("returns ErrAddressNotFound when no default exists", func(t *testing.T) {
		repo := memory.New()
		svc := NewService(repo, metrics.Noop{})

		_, err := svc.GetDefaultAddress(t.Context(), "cust-1")
		require.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrAddressNotFound)
	})

	t.Run("returns ErrEmptyCustomerID for empty customer ID", func(t *testing.T) {
		repo := memory.New()
		svc := NewService(repo, metrics.Noop{})

		_, err := svc.GetDefaultAddress(t.Context(), "")
		require.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrEmptyCustomerID)
	})
}
