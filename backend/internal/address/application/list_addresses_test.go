package application

import (
	"minimart/internal/address/domain"
	"minimart/internal/address/infrastructure/memory"
	"minimart/pkg/metrics"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestListAddresses(t *testing.T) {
	t.Run("returns empty list for customer with no addresses", func(t *testing.T) {
		repo := memory.New()
		svc := NewService(repo, metrics.Noop{})

		got, err := svc.ListAddresses(t.Context(), ListAddressesRequest{
			CustomerID: "cust-1",
		})
		require.NoError(t, err)
		assert.Empty(t, got)
	})

	t.Run("returns all addresses for customer ordered by is_default DESC, created_at DESC", func(t *testing.T) {
		repo := memory.New()
		svc := NewService(repo, metrics.Noop{})

		// Create non-default first
		_, err := svc.CreateAddress(t.Context(), CreateAddressRequest{
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

		// Create default
		_, err = svc.CreateAddress(t.Context(), CreateAddressRequest{
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

		// Create another non-default
		_, err = svc.CreateAddress(t.Context(), CreateAddressRequest{
			CustomerID: "cust-1",
			Label:      "Other",
			FullName:   "John Doe",
			Street:     "789 Other St",
			City:       "Chicago",
			State:      "IL",
			Zip:        "60601",
			Country:    "USA",
			IsDefault:  false,
		})
		require.NoError(t, err)

		got, err := svc.ListAddresses(t.Context(), ListAddressesRequest{
			CustomerID: "cust-1",
		})
		require.NoError(t, err)
		require.Len(t, got, 3)
		// Default should be first
		assert.Equal(t, "Home", got[0].Label)
		assert.True(t, got[0].IsDefault)
		// Then the rest in created_at DESC order
		assert.Equal(t, "Other", got[1].Label)
		assert.Equal(t, "Work", got[2].Label)
	})

	t.Run("returns ErrEmptyCustomerID for empty customer ID", func(t *testing.T) {
		repo := memory.New()
		svc := NewService(repo, metrics.Noop{})

		_, err := svc.ListAddresses(t.Context(), ListAddressesRequest{
			CustomerID: "",
		})
		require.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrEmptyCustomerID)
	})
}
