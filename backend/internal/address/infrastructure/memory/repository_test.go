package memory

import (
	"minimart/internal/address/domain"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMemoryRepository(t *testing.T) {
	t.Run("GetByID returns ErrAddressNotFound for non-existent", func(t *testing.T) {
		repo := New()
		_, err := repo.GetByID(t.Context(), "non-existent")
		require.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrAddressNotFound)
	})

	t.Run("Save and GetByID", func(t *testing.T) {
		repo := New()
		addr := domain.Address{
			ID:         "addr-1",
			CustomerID: "cust-1",
			Label:      "Home",
			FullName:   "John Doe",
			Street:     "123 Main St",
			City:       "New York",
			State:      "NY",
			Zip:        "10001",
			Country:    "USA",
			Version:    1,
		}
		err := repo.Save(t.Context(), addr)
		require.NoError(t, err)

		got, err := repo.GetByID(t.Context(), "addr-1")
		require.NoError(t, err)
		assert.Equal(t, "Home", got.Label)
		assert.Equal(t, "John Doe", got.FullName)
	})

	t.Run("Save returns ErrAddressExists for duplicate ID", func(t *testing.T) {
		repo := New()
		addr := domain.Address{
			ID:         "addr-1",
			CustomerID: "cust-1",
			Label:      "Home",
			FullName:   "John Doe",
			Street:     "123 Main St",
			City:       "New York",
			State:      "NY",
			Zip:        "10001",
			Country:    "USA",
			Version:    1,
		}
		require.NoError(t, repo.Save(t.Context(), addr))

		// Try to save with same ID
		addr2 := addr
		addr2.Label = "Work"
		err := repo.Save(t.Context(), addr2)
		require.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrAddressExists)
	})

	t.Run("GetByCustomerID returns addresses ordered by is_default DESC, created_at DESC", func(t *testing.T) {
		repo := New()
		now := time.Now().UTC()

		// Non-default first (older)
		addr1 := domain.Address{
			ID:         "addr-1",
			CustomerID: "cust-1",
			Label:      "Work",
			FullName:   "John Doe",
			Street:     "456 Office Ave",
			City:       "San Francisco",
			State:      "CA",
			Zip:        "94102",
			Country:    "USA",
			IsDefault:  false,
			CreatedAt:  now.Add(-2 * time.Hour),
			UpdatedAt:  now.Add(-2 * time.Hour),
			Version:    1,
		}

		// Default (newer)
		addr2 := domain.Address{
			ID:         "addr-2",
			CustomerID: "cust-1",
			Label:      "Home",
			FullName:   "John Doe",
			Street:     "123 Main St",
			City:       "New York",
			State:      "NY",
			Zip:        "10001",
			Country:    "USA",
			IsDefault:  true,
			CreatedAt:  now.Add(-1 * time.Hour),
			UpdatedAt:  now.Add(-1 * time.Hour),
			Version:    1,
		}

		// Non-default (newest)
		addr3 := domain.Address{
			ID:         "addr-3",
			CustomerID: "cust-1",
			Label:      "Other",
			FullName:   "John Doe",
			Street:     "789 Other St",
			City:       "Chicago",
			State:      "IL",
			Zip:        "60601",
			Country:    "USA",
			IsDefault:  false,
			CreatedAt:  now,
			UpdatedAt:  now,
			Version:    1,
		}

		require.NoError(t, repo.Save(t.Context(), addr1))
		require.NoError(t, repo.Save(t.Context(), addr2))
		require.NoError(t, repo.Save(t.Context(), addr3))

		got, err := repo.GetByCustomerID(t.Context(), "cust-1")
		require.NoError(t, err)
		require.Len(t, got, 3)
		// Default should be first
		assert.Equal(t, "Home", got[0].Label)
		assert.True(t, got[0].IsDefault)
		// Then newest non-default
		assert.Equal(t, "Other", got[1].Label)
		// Then oldest non-default
		assert.Equal(t, "Work", got[2].Label)
	})

	t.Run("GetDefaultByCustomerID returns default address", func(t *testing.T) {
		repo := New()
		addr := domain.Address{
			ID:         "addr-1",
			CustomerID: "cust-1",
			Label:      "Home",
			FullName:   "John Doe",
			Street:     "123 Main St",
			City:       "New York",
			State:      "NY",
			Zip:        "10001",
			Country:    "USA",
			IsDefault:  true,
			Version:    1,
		}
		require.NoError(t, repo.Save(t.Context(), addr))

		got, err := repo.GetDefaultByCustomerID(t.Context(), "cust-1")
		require.NoError(t, err)
		assert.Equal(t, "Home", got.Label)
		assert.True(t, got.IsDefault)
	})

	t.Run("GetDefaultByCustomerID returns ErrAddressNotFound if no default", func(t *testing.T) {
		repo := New()
		addr := domain.Address{
			ID:         "addr-1",
			CustomerID: "cust-1",
			Label:      "Work",
			FullName:   "John Doe",
			Street:     "456 Office Ave",
			City:       "San Francisco",
			State:      "CA",
			Zip:        "94102",
			Country:    "USA",
			IsDefault:  false,
			Version:    1,
		}
		require.NoError(t, repo.Save(t.Context(), addr))

		_, err := repo.GetDefaultByCustomerID(t.Context(), "cust-1")
		require.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrAddressNotFound)
	})

	t.Run("Update with correct version", func(t *testing.T) {
		repo := New()
		addr := domain.Address{
			ID:         "addr-1",
			CustomerID: "cust-1",
			Label:      "Home",
			FullName:   "John Doe",
			Street:     "123 Main St",
			City:       "New York",
			State:      "NY",
			Zip:        "10001",
			Country:    "USA",
			Version:    1,
		}
		require.NoError(t, repo.Save(t.Context(), addr))

		addr.Label = "Work"
		addr.Version = 1
		updated, err := repo.Update(t.Context(), addr)
		require.NoError(t, err)
		assert.Equal(t, "Work", updated.Label)
		assert.Equal(t, 2, updated.Version)
	})

	t.Run("Update returns ErrInvalidVersion for stale version", func(t *testing.T) {
		repo := New()
		addr := domain.Address{
			ID:         "addr-1",
			CustomerID: "cust-1",
			Label:      "Home",
			FullName:   "John Doe",
			Street:     "123 Main St",
			City:       "New York",
			State:      "NY",
			Zip:        "10001",
			Country:    "USA",
			Version:    1,
		}
		require.NoError(t, repo.Save(t.Context(), addr))

		addr.Label = "Work"
		addr.Version = 1 // stale
		// First update
		_, err := repo.Update(t.Context(), addr)
		require.NoError(t, err)

		// Second update with stale version
		addr.Label = "Work2"
		addr.Version = 1
		_, err = repo.Update(t.Context(), addr)
		require.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrInvalidVersion)
	})

	t.Run("Update returns ErrAddressNotFound for non-existent", func(t *testing.T) {
		repo := New()
		addr := domain.Address{
			ID:         "addr-1",
			CustomerID: "cust-1",
			Label:      "Home",
			FullName:   "John Doe",
			Street:     "123 Main St",
			City:       "New York",
			State:      "NY",
			Zip:        "10001",
			Country:    "USA",
			Version:    1,
		}
		_, err := repo.Update(t.Context(), addr)
		require.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrAddressNotFound)
	})

	t.Run("Delete", func(t *testing.T) {
		repo := New()
		addr := domain.Address{
			ID:         "addr-1",
			CustomerID: "cust-1",
			Label:      "Home",
			FullName:   "John Doe",
			Street:     "123 Main St",
			City:       "New York",
			State:      "NY",
			Zip:        "10001",
			Country:    "USA",
			Version:    1,
		}
		require.NoError(t, repo.Save(t.Context(), addr))

		err := repo.Delete(t.Context(), "addr-1", "cust-1")
		require.NoError(t, err)

		_, err = repo.GetByID(t.Context(), "addr-1")
		require.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrAddressNotFound)
	})

	t.Run("Delete returns ErrAddressNotFound for wrong customer ID", func(t *testing.T) {
		repo := New()
		addr := domain.Address{
			ID:         "addr-1",
			CustomerID: "cust-1",
			Label:      "Home",
			FullName:   "John Doe",
			Street:     "123 Main St",
			City:       "New York",
			State:      "NY",
			Zip:        "10001",
			Country:    "USA",
			Version:    1,
		}
		require.NoError(t, repo.Save(t.Context(), addr))

		err := repo.Delete(t.Context(), "addr-1", "cust-2")
		require.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrAddressNotFound)
	})

	t.Run("SetDefault unsets old default and sets new", func(t *testing.T) {
		repo := New()

		// Create first address as default
		addr1 := domain.Address{
			ID:         "addr-1",
			CustomerID: "cust-1",
			Label:      "Home",
			FullName:   "John Doe",
			Street:     "123 Main St",
			City:       "New York",
			State:      "NY",
			Zip:        "10001",
			Country:    "USA",
			IsDefault:  true,
			Version:    1,
		}
		require.NoError(t, repo.Save(t.Context(), addr1))

		// Create second address
		addr2 := domain.Address{
			ID:         "addr-2",
			CustomerID: "cust-1",
			Label:      "Work",
			FullName:   "John Doe",
			Street:     "456 Office Ave",
			City:       "San Francisco",
			State:      "CA",
			Zip:        "94102",
			Country:    "USA",
			IsDefault:  false,
			Version:    1,
		}
		require.NoError(t, repo.Save(t.Context(), addr2))

		// Set second as default
		updated, err := repo.SetDefault(t.Context(), "addr-2", "cust-1")
		require.NoError(t, err)
		assert.True(t, updated.IsDefault)

		// Verify first is no longer default
		got1, err := repo.GetByID(t.Context(), "addr-1")
		require.NoError(t, err)
		assert.False(t, got1.IsDefault)
	})

	t.Run("SetDefault returns ErrAddressNotFound for wrong customer ID", func(t *testing.T) {
		repo := New()
		addr := domain.Address{
			ID:         "addr-1",
			CustomerID: "cust-1",
			Label:      "Home",
			FullName:   "John Doe",
			Street:     "123 Main St",
			City:       "New York",
			State:      "NY",
			Zip:        "10001",
			Country:    "USA",
			Version:    1,
		}
		require.NoError(t, repo.Save(t.Context(), addr))

		_, err := repo.SetDefault(t.Context(), "addr-1", "cust-2")
		require.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrAddressNotFound)
	})

	t.Run("SetDefault returns ErrAddressNotFound for non-existent ID", func(t *testing.T) {
		repo := New()
		_, err := repo.SetDefault(t.Context(), "non-existent", "cust-1")
		require.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrAddressNotFound)
	})
}
