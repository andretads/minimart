package application

import (
	"minimart/internal/address/domain"
	"minimart/internal/address/infrastructure/memory"
	"minimart/pkg/metrics"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDeleteAddress(t *testing.T) {
	t.Run("deletes an address", func(t *testing.T) {
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

		err = svc.DeleteAddress(t.Context(), DeleteAddressRequest{
			ID:         created.ID,
			CustomerID: "cust-1",
		})
		require.NoError(t, err)

		_, err = svc.GetAddress(t.Context(), created.ID, "cust-1")
		require.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrAddressNotFound)
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

		err := svc.DeleteAddress(t.Context(), DeleteAddressRequest{
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

		err := svc.DeleteAddress(t.Context(), DeleteAddressRequest{
			ID:         created.ID,
			CustomerID: "cust-2",
		})
		require.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrAddressNotFound)
	})

	t.Run("returns ErrAddressNotFound for non-existent ID", func(t *testing.T) {
		repo := memory.New()
		svc := NewService(repo, metrics.Noop{})

		err := svc.DeleteAddress(t.Context(), DeleteAddressRequest{
			ID:         "non-existent",
			CustomerID: "cust-1",
		})
		require.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrAddressNotFound)
	})
}
