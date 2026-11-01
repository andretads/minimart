package application

import (
	"minimart/internal/address/domain"
	"minimart/internal/address/infrastructure/memory"
	"minimart/pkg/metrics"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUpdateAddress(t *testing.T) {
	t.Run("updates an address", func(t *testing.T) {
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

		updated, err := svc.UpdateAddress(t.Context(), UpdateAddressRequest{
			ID:         created.ID,
			CustomerID: "cust-1",
			Label:      "Work",
			FullName:   "Jane Smith",
			Street:     "456 Office Ave",
			City:       "San Francisco",
			State:      "CA",
			Zip:        "94102",
			Country:    "USA",
			Phone:      "+1-555-5678",
			Version:    created.Version,
		})
		require.NoError(t, err)
		assert.Equal(t, "Work", updated.Label)
		assert.Equal(t, "Jane Smith", updated.FullName)
		assert.Equal(t, "456 Office Ave", updated.Street)
		assert.Equal(t, "San Francisco", updated.City)
		assert.Equal(t, "CA", updated.State)
		assert.Equal(t, "94102", updated.Zip)
		assert.Equal(t, "USA", updated.Country)
		assert.Equal(t, "+1-555-5678", updated.Phone)
		assert.Equal(t, 2, updated.Version)
	})

	t.Run("rejects empty customer ID", func(t *testing.T) {
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

		_, err := svc.UpdateAddress(t.Context(), UpdateAddressRequest{
			ID:         created.ID,
			CustomerID: "",
			Label:      "Work",
			FullName:   "Jane Smith",
			Street:     "456 Office Ave",
			City:       "San Francisco",
			State:      "CA",
			Zip:        "94102",
			Country:    "USA",
			Version:    created.Version,
		})
		require.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrEmptyCustomerID)
	})

	t.Run("rejects invalid version", func(t *testing.T) {
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

		_, err := svc.UpdateAddress(t.Context(), UpdateAddressRequest{
			ID:         created.ID,
			CustomerID: "cust-1",
			Label:      "Work",
			FullName:   "Jane Smith",
			Street:     "456 Office Ave",
			City:       "San Francisco",
			State:      "CA",
			Zip:        "94102",
			Country:    "USA",
			Version:    0,
		})
		require.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrInvalidVersion)
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

		_, err := svc.UpdateAddress(t.Context(), UpdateAddressRequest{
			ID:         created.ID,
			CustomerID: "cust-2",
			Label:      "Work",
			FullName:   "Jane Smith",
			Street:     "456 Office Ave",
			City:       "San Francisco",
			State:      "CA",
			Zip:        "94102",
			Country:    "USA",
			Version:    created.Version,
		})
		require.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrAddressNotFound)
	})

	t.Run("returns ErrInvalidVersion for stale version", func(t *testing.T) {
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

		// First update
		_, err := svc.UpdateAddress(t.Context(), UpdateAddressRequest{
			ID:         created.ID,
			CustomerID: "cust-1",
			Label:      "Work",
			FullName:   "Jane Smith",
			Street:     "456 Office Ave",
			City:       "San Francisco",
			State:      "CA",
			Zip:        "94102",
			Country:    "USA",
			Version:    created.Version,
		})
		require.NoError(t, err)

		// Second update with stale version
		_, err = svc.UpdateAddress(t.Context(), UpdateAddressRequest{
			ID:         created.ID,
			CustomerID: "cust-1",
			Label:      "Work2",
			FullName:   "Jane Smith",
			Street:     "456 Office Ave",
			City:       "San Francisco",
			State:      "CA",
			Zip:        "94102",
			Country:    "USA",
			Version:    created.Version,
		})
		require.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrInvalidVersion)
	})

	t.Run("rejects empty label", func(t *testing.T) {
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

		_, err := svc.UpdateAddress(t.Context(), UpdateAddressRequest{
			ID:         created.ID,
			CustomerID: "cust-1",
			Label:      "   ",
			FullName:   "Jane Smith",
			Street:     "456 Office Ave",
			City:       "San Francisco",
			State:      "CA",
			Zip:        "94102",
			Country:    "USA",
			Version:    created.Version,
		})
		require.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrEmptyLabel)
	})

	t.Run("rejects label too long", func(t *testing.T) {
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

		longLabel := strings.Repeat("a", 51)
		_, err := svc.UpdateAddress(t.Context(), UpdateAddressRequest{
			ID:         created.ID,
			CustomerID: "cust-1",
			Label:      longLabel,
			FullName:   "Jane Smith",
			Street:     "456 Office Ave",
			City:       "San Francisco",
			State:      "CA",
			Zip:        "94102",
			Country:    "USA",
			Version:    created.Version,
		})
		require.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrLabelTooLong)
	})

	t.Run("rejects empty full name", func(t *testing.T) {
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

		_, err := svc.UpdateAddress(t.Context(), UpdateAddressRequest{
			ID:         created.ID,
			CustomerID: "cust-1",
			Label:      "Work",
			FullName:   "   ",
			Street:     "456 Office Ave",
			City:       "San Francisco",
			State:      "CA",
			Zip:        "94102",
			Country:    "USA",
			Version:    created.Version,
		})
		require.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrEmptyFullName)
	})

	t.Run("trims whitespace from all fields", func(t *testing.T) {
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

		updated, err := svc.UpdateAddress(t.Context(), UpdateAddressRequest{
			ID:         created.ID,
			CustomerID: "cust-1",
			Label:      "  Work  ",
			FullName:   "  Jane Smith  ",
			Street:     "  456 Office Ave  ",
			City:       "  San Francisco  ",
			State:      "  CA  ",
			Zip:        "  94102  ",
			Country:    "  USA  ",
			Phone:      "  +1-555-5678  ",
			Version:    created.Version,
		})
		require.NoError(t, err)
		assert.Equal(t, "Work", updated.Label)
		assert.Equal(t, "Jane Smith", updated.FullName)
		assert.Equal(t, "456 Office Ave", updated.Street)
		assert.Equal(t, "San Francisco", updated.City)
		assert.Equal(t, "CA", updated.State)
		assert.Equal(t, "94102", updated.Zip)
		assert.Equal(t, "USA", updated.Country)
		assert.Equal(t, "+1-555-5678", updated.Phone)
	})
}
