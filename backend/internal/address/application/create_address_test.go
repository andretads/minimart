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

func TestCreateAddress(t *testing.T) {
	t.Run("creates an address with all fields", func(t *testing.T) {
		repo := memory.New()
		svc := NewService(repo, metrics.Noop{})

		got, err := svc.CreateAddress(t.Context(), CreateAddressRequest{
			CustomerID: "cust-1",
			Label:      "Home",
			FullName:   "John Doe",
			Street:     "123 Main St",
			City:       "New York",
			State:      "NY",
			Zip:        "10001",
			Country:    "USA",
			Phone:      "+1-555-1234",
			IsDefault:  false,
		})
		require.NoError(t, err)
		assert.NotEmpty(t, got.ID)
		assert.Equal(t, "cust-1", got.CustomerID)
		assert.Equal(t, "Home", got.Label)
		assert.Equal(t, "John Doe", got.FullName)
		assert.Equal(t, "123 Main St", got.Street)
		assert.Equal(t, "New York", got.City)
		assert.Equal(t, "NY", got.State)
		assert.Equal(t, "10001", got.Zip)
		assert.Equal(t, "USA", got.Country)
		assert.Equal(t, "+1-555-1234", got.Phone)
		assert.False(t, got.IsDefault)
		assert.Equal(t, 1, got.Version)
	})

	t.Run("creates an address without phone", func(t *testing.T) {
		repo := memory.New()
		svc := NewService(repo, metrics.Noop{})

		got, err := svc.CreateAddress(t.Context(), CreateAddressRequest{
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
		assert.Empty(t, got.Phone)
	})

	t.Run("creates an address as default", func(t *testing.T) {
		repo := memory.New()
		svc := NewService(repo, metrics.Noop{})

		got, err := svc.CreateAddress(t.Context(), CreateAddressRequest{
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
		assert.True(t, got.IsDefault)
	})

	t.Run("rejects empty customer ID", func(t *testing.T) {
		repo := memory.New()
		svc := NewService(repo, metrics.Noop{})

		_, err := svc.CreateAddress(t.Context(), CreateAddressRequest{
			Label:    "Home",
			FullName: "John Doe",
			Street:   "123 Main St",
			City:     "New York",
			State:    "NY",
			Zip:      "10001",
			Country:  "USA",
		})
		require.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrEmptyCustomerID)
	})

	t.Run("rejects empty label", func(t *testing.T) {
		repo := memory.New()
		svc := NewService(repo, metrics.Noop{})

		_, err := svc.CreateAddress(t.Context(), CreateAddressRequest{
			CustomerID: "cust-1",
			Label:      "   ",
			FullName:   "John Doe",
			Street:     "123 Main St",
			City:       "New York",
			State:      "NY",
			Zip:        "10001",
			Country:    "USA",
		})
		require.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrEmptyLabel)
	})

	t.Run("rejects label too long", func(t *testing.T) {
		repo := memory.New()
		svc := NewService(repo, metrics.Noop{})

		longLabel := strings.Repeat("a", 51)
		_, err := svc.CreateAddress(t.Context(), CreateAddressRequest{
			CustomerID: "cust-1",
			Label:      longLabel,
			FullName:   "John Doe",
			Street:     "123 Main St",
			City:       "New York",
			State:      "NY",
			Zip:        "10001",
			Country:    "USA",
		})
		require.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrLabelTooLong)
	})

	t.Run("rejects empty full name", func(t *testing.T) {
		repo := memory.New()
		svc := NewService(repo, metrics.Noop{})

		_, err := svc.CreateAddress(t.Context(), CreateAddressRequest{
			CustomerID: "cust-1",
			Label:      "Home",
			FullName:   "   ",
			Street:     "123 Main St",
			City:       "New York",
			State:      "NY",
			Zip:        "10001",
			Country:    "USA",
		})
		require.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrEmptyFullName)
	})

	t.Run("rejects full name too long", func(t *testing.T) {
		repo := memory.New()
		svc := NewService(repo, metrics.Noop{})

		longName := strings.Repeat("a", 101)
		_, err := svc.CreateAddress(t.Context(), CreateAddressRequest{
			CustomerID: "cust-1",
			Label:      "Home",
			FullName:   longName,
			Street:     "123 Main St",
			City:       "New York",
			State:      "NY",
			Zip:        "10001",
			Country:    "USA",
		})
		require.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrFullNameTooLong)
	})

	t.Run("rejects empty street", func(t *testing.T) {
		repo := memory.New()
		svc := NewService(repo, metrics.Noop{})

		_, err := svc.CreateAddress(t.Context(), CreateAddressRequest{
			CustomerID: "cust-1",
			Label:      "Home",
			FullName:   "John Doe",
			Street:     "   ",
			City:       "New York",
			State:      "NY",
			Zip:        "10001",
			Country:    "USA",
		})
		require.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrEmptyStreet)
	})

	t.Run("rejects street too long", func(t *testing.T) {
		repo := memory.New()
		svc := NewService(repo, metrics.Noop{})

		longStreet := strings.Repeat("a", 201)
		_, err := svc.CreateAddress(t.Context(), CreateAddressRequest{
			CustomerID: "cust-1",
			Label:      "Home",
			FullName:   "John Doe",
			Street:     longStreet,
			City:       "New York",
			State:      "NY",
			Zip:        "10001",
			Country:    "USA",
		})
		require.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrStreetTooLong)
	})

	t.Run("rejects empty city", func(t *testing.T) {
		repo := memory.New()
		svc := NewService(repo, metrics.Noop{})

		_, err := svc.CreateAddress(t.Context(), CreateAddressRequest{
			CustomerID: "cust-1",
			Label:      "Home",
			FullName:   "John Doe",
			Street:     "123 Main St",
			City:       "   ",
			State:      "NY",
			Zip:        "10001",
			Country:    "USA",
		})
		require.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrEmptyCity)
	})

	t.Run("rejects city too long", func(t *testing.T) {
		repo := memory.New()
		svc := NewService(repo, metrics.Noop{})

		longCity := strings.Repeat("a", 101)
		_, err := svc.CreateAddress(t.Context(), CreateAddressRequest{
			CustomerID: "cust-1",
			Label:      "Home",
			FullName:   "John Doe",
			Street:     "123 Main St",
			City:       longCity,
			State:      "NY",
			Zip:        "10001",
			Country:    "USA",
		})
		require.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrCityTooLong)
	})

	t.Run("rejects empty state", func(t *testing.T) {
		repo := memory.New()
		svc := NewService(repo, metrics.Noop{})

		_, err := svc.CreateAddress(t.Context(), CreateAddressRequest{
			CustomerID: "cust-1",
			Label:      "Home",
			FullName:   "John Doe",
			Street:     "123 Main St",
			City:       "New York",
			State:      "   ",
			Zip:        "10001",
			Country:    "USA",
		})
		require.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrEmptyState)
	})

	t.Run("rejects state too long", func(t *testing.T) {
		repo := memory.New()
		svc := NewService(repo, metrics.Noop{})

		longState := strings.Repeat("a", 101)
		_, err := svc.CreateAddress(t.Context(), CreateAddressRequest{
			CustomerID: "cust-1",
			Label:      "Home",
			FullName:   "John Doe",
			Street:     "123 Main St",
			City:       "New York",
			State:      longState,
			Zip:        "10001",
			Country:    "USA",
		})
		require.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrStateTooLong)
	})

	t.Run("rejects empty zip", func(t *testing.T) {
		repo := memory.New()
		svc := NewService(repo, metrics.Noop{})

		_, err := svc.CreateAddress(t.Context(), CreateAddressRequest{
			CustomerID: "cust-1",
			Label:      "Home",
			FullName:   "John Doe",
			Street:     "123 Main St",
			City:       "New York",
			State:      "NY",
			Zip:        "   ",
			Country:    "USA",
		})
		require.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrEmptyZip)
	})

	t.Run("rejects zip too long", func(t *testing.T) {
		repo := memory.New()
		svc := NewService(repo, metrics.Noop{})

		longZip := strings.Repeat("a", 21)
		_, err := svc.CreateAddress(t.Context(), CreateAddressRequest{
			CustomerID: "cust-1",
			Label:      "Home",
			FullName:   "John Doe",
			Street:     "123 Main St",
			City:       "New York",
			State:      "NY",
			Zip:        longZip,
			Country:    "USA",
		})
		require.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrZipTooLong)
	})

	t.Run("rejects empty country", func(t *testing.T) {
		repo := memory.New()
		svc := NewService(repo, metrics.Noop{})

		_, err := svc.CreateAddress(t.Context(), CreateAddressRequest{
			CustomerID: "cust-1",
			Label:      "Home",
			FullName:   "John Doe",
			Street:     "123 Main St",
			City:       "New York",
			State:      "NY",
			Zip:        "10001",
			Country:    "   ",
		})
		require.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrEmptyCountry)
	})

	t.Run("rejects country too long", func(t *testing.T) {
		repo := memory.New()
		svc := NewService(repo, metrics.Noop{})

		longCountry := strings.Repeat("a", 101)
		_, err := svc.CreateAddress(t.Context(), CreateAddressRequest{
			CustomerID: "cust-1",
			Label:      "Home",
			FullName:   "John Doe",
			Street:     "123 Main St",
			City:       "New York",
			State:      "NY",
			Zip:        "10001",
			Country:    longCountry,
		})
		require.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrCountryTooLong)
	})

	t.Run("rejects phone too long", func(t *testing.T) {
		repo := memory.New()
		svc := NewService(repo, metrics.Noop{})

		longPhone := strings.Repeat("a", 31)
		_, err := svc.CreateAddress(t.Context(), CreateAddressRequest{
			CustomerID: "cust-1",
			Label:      "Home",
			FullName:   "John Doe",
			Street:     "123 Main St",
			City:       "New York",
			State:      "NY",
			Zip:        "10001",
			Country:    "USA",
			Phone:      longPhone,
		})
		require.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrPhoneTooLong)
	})

	t.Run("trims whitespace from all fields", func(t *testing.T) {
		repo := memory.New()
		svc := NewService(repo, metrics.Noop{})

		got, err := svc.CreateAddress(t.Context(), CreateAddressRequest{
			CustomerID: "cust-1",
			Label:      "  Home  ",
			FullName:   "  John Doe  ",
			Street:     "  123 Main St  ",
			City:       "  New York  ",
			State:      "  NY  ",
			Zip:        "  10001  ",
			Country:    "  USA  ",
			Phone:      "  +1-555-1234  ",
		})
		require.NoError(t, err)
		assert.Equal(t, "Home", got.Label)
		assert.Equal(t, "John Doe", got.FullName)
		assert.Equal(t, "123 Main St", got.Street)
		assert.Equal(t, "New York", got.City)
		assert.Equal(t, "NY", got.State)
		assert.Equal(t, "10001", got.Zip)
		assert.Equal(t, "USA", got.Country)
		assert.Equal(t, "+1-555-1234", got.Phone)
	})
}
