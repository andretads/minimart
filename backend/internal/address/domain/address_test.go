package domain

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// testFields returns a valid address field set; each subtest mutates the one
// field under test.
func testFields() AddressFields {
	return AddressFields{
		Label:    "Home",
		FullName: "John Doe",
		Street:   "123 Main St",
		City:     "New York",
		State:    "NY",
		Zip:      "10001",
		Country:  "USA",
		Phone:    "+1-555-1234",
	}
}

// mustAddress returns a persisted-shaped address built from testFields.
func mustAddress(t *testing.T) Address {
	t.Helper()
	a, err := NewAddress("addr-1", "cust-1", testFields(), false)
	require.NoError(t, err)
	return a
}

func TestNewAddress(t *testing.T) {
	t.Run("creates an address with all fields", func(t *testing.T) {
		a, err := NewAddress("addr-1", "cust-1", testFields(), false)
		require.NoError(t, err)
		assert.Equal(t, "addr-1", a.ID)
		assert.Equal(t, "cust-1", a.CustomerID)
		assert.Equal(t, "Home", a.Label)
		assert.Equal(t, "John Doe", a.FullName)
		assert.Equal(t, "123 Main St", a.Street)
		assert.Equal(t, "New York", a.City)
		assert.Equal(t, "NY", a.State)
		assert.Equal(t, "10001", a.Zip)
		assert.Equal(t, "USA", a.Country)
		assert.Equal(t, "+1-555-1234", a.Phone)
		assert.False(t, a.IsDefault)
		assert.Equal(t, 1, a.Version)
		assert.False(t, a.CreatedAt.IsZero())
		assert.False(t, a.UpdatedAt.IsZero())
	})

	t.Run("creates an address with default flag", func(t *testing.T) {
		a, err := NewAddress("addr-1", "cust-1", testFields(), true)
		require.NoError(t, err)
		assert.True(t, a.IsDefault)
	})

	t.Run("creates an address without phone", func(t *testing.T) {
		fields := testFields()
		fields.Phone = ""
		a, err := NewAddress("addr-1", "cust-1", fields, false)
		require.NoError(t, err)
		assert.Empty(t, a.Phone)
	})

	t.Run("rejects empty customer ID", func(t *testing.T) {
		_, err := NewAddress("addr-1", "", testFields(), false)
		require.Error(t, err)
		assert.ErrorIs(t, err, ErrEmptyCustomerID)
	})

	t.Run("rejects empty label", func(t *testing.T) {
		fields := testFields()
		fields.Label = "   "
		_, err := NewAddress("addr-1", "cust-1", fields, false)
		require.Error(t, err)
		assert.ErrorIs(t, err, ErrEmptyLabel)
	})

	t.Run("rejects label too long", func(t *testing.T) {
		fields := testFields()
		fields.Label = strings.Repeat("a", 51)
		_, err := NewAddress("addr-1", "cust-1", fields, false)
		require.Error(t, err)
		assert.ErrorIs(t, err, ErrLabelTooLong)
	})

	t.Run("rejects empty full name", func(t *testing.T) {
		fields := testFields()
		fields.FullName = "   "
		_, err := NewAddress("addr-1", "cust-1", fields, false)
		require.Error(t, err)
		assert.ErrorIs(t, err, ErrEmptyFullName)
	})

	t.Run("rejects full name too long", func(t *testing.T) {
		fields := testFields()
		fields.FullName = strings.Repeat("a", 101)
		_, err := NewAddress("addr-1", "cust-1", fields, false)
		require.Error(t, err)
		assert.ErrorIs(t, err, ErrFullNameTooLong)
	})

	t.Run("rejects empty street", func(t *testing.T) {
		fields := testFields()
		fields.Street = "   "
		_, err := NewAddress("addr-1", "cust-1", fields, false)
		require.Error(t, err)
		assert.ErrorIs(t, err, ErrEmptyStreet)
	})

	t.Run("rejects street too long", func(t *testing.T) {
		fields := testFields()
		fields.Street = strings.Repeat("a", 201)
		_, err := NewAddress("addr-1", "cust-1", fields, false)
		require.Error(t, err)
		assert.ErrorIs(t, err, ErrStreetTooLong)
	})

	t.Run("rejects empty city", func(t *testing.T) {
		fields := testFields()
		fields.City = "   "
		_, err := NewAddress("addr-1", "cust-1", fields, false)
		require.Error(t, err)
		assert.ErrorIs(t, err, ErrEmptyCity)
	})

	t.Run("rejects city too long", func(t *testing.T) {
		fields := testFields()
		fields.City = strings.Repeat("a", 101)
		_, err := NewAddress("addr-1", "cust-1", fields, false)
		require.Error(t, err)
		assert.ErrorIs(t, err, ErrCityTooLong)
	})

	t.Run("rejects empty state", func(t *testing.T) {
		fields := testFields()
		fields.State = "   "
		_, err := NewAddress("addr-1", "cust-1", fields, false)
		require.Error(t, err)
		assert.ErrorIs(t, err, ErrEmptyState)
	})

	t.Run("rejects state too long", func(t *testing.T) {
		fields := testFields()
		fields.State = strings.Repeat("a", 101)
		_, err := NewAddress("addr-1", "cust-1", fields, false)
		require.Error(t, err)
		assert.ErrorIs(t, err, ErrStateTooLong)
	})

	t.Run("rejects empty zip", func(t *testing.T) {
		fields := testFields()
		fields.Zip = "   "
		_, err := NewAddress("addr-1", "cust-1", fields, false)
		require.Error(t, err)
		assert.ErrorIs(t, err, ErrEmptyZip)
	})

	t.Run("rejects zip too long", func(t *testing.T) {
		fields := testFields()
		fields.Zip = strings.Repeat("a", 21)
		_, err := NewAddress("addr-1", "cust-1", fields, false)
		require.Error(t, err)
		assert.ErrorIs(t, err, ErrZipTooLong)
	})

	t.Run("rejects empty country", func(t *testing.T) {
		fields := testFields()
		fields.Country = "   "
		_, err := NewAddress("addr-1", "cust-1", fields, false)
		require.Error(t, err)
		assert.ErrorIs(t, err, ErrEmptyCountry)
	})

	t.Run("rejects country too long", func(t *testing.T) {
		fields := testFields()
		fields.Country = strings.Repeat("a", 101)
		_, err := NewAddress("addr-1", "cust-1", fields, false)
		require.Error(t, err)
		assert.ErrorIs(t, err, ErrCountryTooLong)
	})

	t.Run("rejects phone too long", func(t *testing.T) {
		fields := testFields()
		fields.Phone = strings.Repeat("a", 31)
		_, err := NewAddress("addr-1", "cust-1", fields, false)
		require.Error(t, err)
		assert.ErrorIs(t, err, ErrPhoneTooLong)
	})

	t.Run("trims whitespace from all fields", func(t *testing.T) {
		fields := AddressFields{
			Label:    "  Home  ",
			FullName: "  John Doe  ",
			Street:   "  123 Main St  ",
			City:     "  New York  ",
			State:    "  NY  ",
			Zip:      "  10001  ",
			Country:  "  USA  ",
			Phone:    "  +1-555-1234  ",
		}
		a, err := NewAddress("addr-1", "cust-1", fields, false)
		require.NoError(t, err)
		assert.Equal(t, "Home", a.Label)
		assert.Equal(t, "John Doe", a.FullName)
		assert.Equal(t, "123 Main St", a.Street)
		assert.Equal(t, "New York", a.City)
		assert.Equal(t, "NY", a.State)
		assert.Equal(t, "10001", a.Zip)
		assert.Equal(t, "USA", a.Country)
		assert.Equal(t, "+1-555-1234", a.Phone)
	})
}

func TestAddress_UpdateDetails(t *testing.T) {
	updated := AddressFields{
		Label:    "Work",
		FullName: "Jane Smith",
		Street:   "456 Office Ave",
		City:     "San Francisco",
		State:    "CA",
		Zip:      "94102",
		Country:  "USA",
		Phone:    "+1-555-5678",
	}

	t.Run("updates all fields", func(t *testing.T) {
		a := mustAddress(t)
		err := a.UpdateDetails(updated)
		require.NoError(t, err)
		assert.Equal(t, "Work", a.Label)
		assert.Equal(t, "Jane Smith", a.FullName)
		assert.Equal(t, "456 Office Ave", a.Street)
		assert.Equal(t, "San Francisco", a.City)
		assert.Equal(t, "CA", a.State)
		assert.Equal(t, "94102", a.Zip)
		assert.Equal(t, "USA", a.Country)
		assert.Equal(t, "+1-555-5678", a.Phone)
		assert.False(t, a.UpdatedAt.IsZero())
		assert.Equal(t, 1, a.Version) // version not bumped by domain
	})

	t.Run("rejects empty label", func(t *testing.T) {
		a := mustAddress(t)
		fields := updated
		fields.Label = "   "
		err := a.UpdateDetails(fields)
		require.Error(t, err)
		assert.ErrorIs(t, err, ErrEmptyLabel)
	})

	t.Run("rejects empty full name", func(t *testing.T) {
		a := mustAddress(t)
		fields := updated
		fields.FullName = "   "
		err := a.UpdateDetails(fields)
		require.Error(t, err)
		assert.ErrorIs(t, err, ErrEmptyFullName)
	})

	t.Run("rejects empty street", func(t *testing.T) {
		a := mustAddress(t)
		fields := updated
		fields.Street = "   "
		err := a.UpdateDetails(fields)
		require.Error(t, err)
		assert.ErrorIs(t, err, ErrEmptyStreet)
	})

	t.Run("rejects empty city", func(t *testing.T) {
		a := mustAddress(t)
		fields := updated
		fields.City = "   "
		err := a.UpdateDetails(fields)
		require.Error(t, err)
		assert.ErrorIs(t, err, ErrEmptyCity)
	})

	t.Run("rejects empty state", func(t *testing.T) {
		a := mustAddress(t)
		fields := updated
		fields.State = "   "
		err := a.UpdateDetails(fields)
		require.Error(t, err)
		assert.ErrorIs(t, err, ErrEmptyState)
	})

	t.Run("rejects empty zip", func(t *testing.T) {
		a := mustAddress(t)
		fields := updated
		fields.Zip = "   "
		err := a.UpdateDetails(fields)
		require.Error(t, err)
		assert.ErrorIs(t, err, ErrEmptyZip)
	})

	t.Run("rejects empty country", func(t *testing.T) {
		a := mustAddress(t)
		fields := updated
		fields.Country = "   "
		err := a.UpdateDetails(fields)
		require.Error(t, err)
		assert.ErrorIs(t, err, ErrEmptyCountry)
	})

	t.Run("rejects phone too long", func(t *testing.T) {
		a := mustAddress(t)
		fields := updated
		fields.Phone = strings.Repeat("a", 31)
		err := a.UpdateDetails(fields)
		require.Error(t, err)
		assert.ErrorIs(t, err, ErrPhoneTooLong)
	})

	t.Run("trims whitespace from all fields", func(t *testing.T) {
		a := mustAddress(t)
		fields := AddressFields{
			Label:    "  Work  ",
			FullName: "  Jane Smith  ",
			Street:   "  456 Office Ave  ",
			City:     "  San Francisco  ",
			State:    "  CA  ",
			Zip:      "  94102  ",
			Country:  "  USA  ",
			Phone:    "  +1-555-5678  ",
		}
		err := a.UpdateDetails(fields)
		require.NoError(t, err)
		assert.Equal(t, "Work", a.Label)
		assert.Equal(t, "Jane Smith", a.FullName)
		assert.Equal(t, "456 Office Ave", a.Street)
		assert.Equal(t, "San Francisco", a.City)
		assert.Equal(t, "CA", a.State)
		assert.Equal(t, "94102", a.Zip)
		assert.Equal(t, "USA", a.Country)
		assert.Equal(t, "+1-555-5678", a.Phone)
	})
}

func TestAddress_SetDefault(t *testing.T) {
	t.Run("sets default flag", func(t *testing.T) {
		a := mustAddress(t)
		a.SetDefault()
		assert.True(t, a.IsDefault)
		assert.False(t, a.UpdatedAt.IsZero())
	})

	t.Run("idempotent - no change if already default", func(t *testing.T) {
		a, err := NewAddress("addr-1", "cust-1", testFields(), true)
		require.NoError(t, err)
		updatedAt := a.UpdatedAt
		a.SetDefault()
		assert.True(t, a.IsDefault)
		assert.Equal(t, updatedAt, a.UpdatedAt) // unchanged
	})
}

func TestAddress_UnsetDefault(t *testing.T) {
	t.Run("unsets default flag", func(t *testing.T) {
		a, err := NewAddress("addr-1", "cust-1", testFields(), true)
		require.NoError(t, err)
		a.UnsetDefault()
		assert.False(t, a.IsDefault)
		assert.False(t, a.UpdatedAt.IsZero())
	})

	t.Run("idempotent - no change if already not default", func(t *testing.T) {
		a := mustAddress(t)
		updatedAt := a.UpdatedAt
		a.UnsetDefault()
		assert.False(t, a.IsDefault)
		assert.Equal(t, updatedAt, a.UpdatedAt) // unchanged
	})
}

func TestAddress_IsDefaultAddress(t *testing.T) {
	t.Run("returns true for default address", func(t *testing.T) {
		a, err := NewAddress("addr-1", "cust-1", testFields(), true)
		require.NoError(t, err)
		assert.True(t, a.IsDefaultAddress())
	})

	t.Run("returns false for non-default address", func(t *testing.T) {
		a := mustAddress(t)
		assert.False(t, a.IsDefaultAddress())
	})
}

func TestNewAddress_NormalizesCountry(t *testing.T) {
	newWithCountry := func(t *testing.T, country string) Address {
		t.Helper()
		fields := testFields()
		fields.Phone = ""
		fields.Country = country
		a, err := NewAddress("addr-1", "cust-1", fields, false)
		require.NoError(t, err)
		return a
	}

	assert.Equal(t, "BR", newWithCountry(t, "Brasil").Country)
	assert.Equal(t, "BR", newWithCountry(t, "  br  ").Country)

	// unknown values are preserved (domain-specific fixtures)
	assert.Equal(t, "C-1234", newWithCountry(t, "C-1234").Country)
}
