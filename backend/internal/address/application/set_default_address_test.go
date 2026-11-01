package application

import (
	"minimart/internal/address/domain"
	"minimart/internal/address/infrastructure/memory"
	"minimart/pkg/metrics"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSetDefaultAddress(t *testing.T) {
	t.Run("sets address as default", func(t *testing.T) {
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
			IsDefault:  false,
		})
		require.NoError(t, err)

		updated, err := svc.SetDefaultAddress(t.Context(), SetDefaultAddressRequest{
			ID:         created.ID,
			CustomerID: "cust-1",
		})
		require.NoError(t, err)
		assert.True(t, updated.IsDefault)
		assert.Equal(t, 2, updated.Version)
	})

	t.Run("unsets old default when setting new default", func(t *testing.T) {
		repo := memory.New()
		svc := NewService(repo, metrics.Noop{})

		// Create first address as default
		first, err := svc.CreateAddress(t.Context(), CreateAddressRequest{
			CustomerID: "cust-1",
			Label:      "Home",
			FullName:   "John Doe",
			Street:     "123 Main St",
			City:       "New York",
			State:      "NY",
			Zip:        "10001",
			Country:    "USA",
			IsDefault:  true,
		})
		require.NoError(t, err)
		assert.True(t, first.IsDefault)

		// Create second address as non-default
		second, err := svc.CreateAddress(t.Context(), CreateAddressRequest{
			CustomerID: "cust-1",
			Label:      "Work",
			FullName:   "John Doe",
			Street:     "456 Office Ave",
			City:       "San Francisco",
			State:      "CA",
			Zip:        "94102",
			Country:    "USA",
			IsDefault:  false,
		})
		require.NoError(t, err)
		assert.False(t, second.IsDefault)

		// Set second as default
		updated, err := svc.SetDefaultAddress(t.Context(), SetDefaultAddressRequest{
			ID:         second.ID,
			CustomerID: "cust-1",
		})
		require.NoError(t, err)
		assert.True(t, updated.IsDefault)

		// Verify first is no longer default
		firstUpdated, err := svc.GetAddress(t.Context(), first.ID, "cust-1")
		require.NoError(t, err)
		assert.False(t, firstUpdated.IsDefault)
	})

	t.Run("returns ErrEmptyCustomerID for empty customer ID", func(t *testing.T) {
		repo := memory.New()
		svc := NewService(repo, metrics.Noop{})

		created, _ := svc.CreateAddress(t.Context(), CreateAddressRequest{
			CustomerID: "cust-1",
			Label:      "Home",
			FullName:   "John Doe",
			Street:     "123 Main St",
			City:       "New York",
			State:      "NY",
			Zip:        "10001",
			Country:    "USA",
		})

		_, err := svc.SetDefaultAddress(t.Context(), SetDefaultAddressRequest{
			ID:         created.ID,
			CustomerID: "",
		})
		require.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrEmptyCustomerID)
	})

	t.Run("returns ErrAddressNotFound for wrong customer ID", func(t *testing.T) {
		repo := memory.New()
		svc := NewService(repo, metrics.Noop{})

		created, _ := svc.CreateAddress(t.Context(), CreateAddressRequest{
			CustomerID: "cust-1",
			Label:      "Home",
			FullName:   "John Doe",
			Street:     "123 Main St",
			City:       "New York",
			State:      "NY",
			Zip:        "10001",
			Country:    "USA",
		})

		_, err := svc.SetDefaultAddress(t.Context(), SetDefaultAddressRequest{
			ID:         created.ID,
			CustomerID: "cust-2",
		})
		require.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrAddressNotFound)
	})

	t.Run("returns ErrAddressNotFound for non-existent ID", func(t *testing.T) {
		repo := memory.New()
		svc := NewService(repo, metrics.Noop{})

		_, err := svc.SetDefaultAddress(t.Context(), SetDefaultAddressRequest{
			ID:         "non-existent",
			CustomerID: "cust-1",
		})
		require.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrAddressNotFound)
	})
}
