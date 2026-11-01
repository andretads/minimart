//go:build integration

package test

import (
	"minimart/internal/address/domain"
	"strings"
	"testing"

	addressapp "minimart/internal/address/application"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCreateAddressIntegration(t *testing.T) {
	svc, customerSvc := newAddressServices(t)

	t.Run("Empty customer ID is rejected", func(t *testing.T) {
		_, err := svc.CreateAddress(t.Context(), validCreateAddressRequest(""))
		require.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrEmptyCustomerID)
	})

	t.Run("Empty label is rejected", func(t *testing.T) {
		req := validCreateAddressRequest(newTestCustomer(t, customerSvc))
		req.Label = "   "
		_, err := svc.CreateAddress(t.Context(), req)
		require.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrEmptyLabel)
	})

	t.Run("Label too long is rejected", func(t *testing.T) {
		req := validCreateAddressRequest(newTestCustomer(t, customerSvc))
		req.Label = strings.Repeat("a", 51)
		_, err := svc.CreateAddress(t.Context(), req)
		require.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrLabelTooLong)
	})

	t.Run("Empty full name is rejected", func(t *testing.T) {
		req := validCreateAddressRequest(newTestCustomer(t, customerSvc))
		req.FullName = "   "
		_, err := svc.CreateAddress(t.Context(), req)
		require.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrEmptyFullName)
	})

	t.Run("Full name too long is rejected", func(t *testing.T) {
		req := validCreateAddressRequest(newTestCustomer(t, customerSvc))
		req.FullName = strings.Repeat("a", 101)
		_, err := svc.CreateAddress(t.Context(), req)
		require.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrFullNameTooLong)
	})

	t.Run("Empty street is rejected", func(t *testing.T) {
		req := validCreateAddressRequest(newTestCustomer(t, customerSvc))
		req.Street = "   "
		_, err := svc.CreateAddress(t.Context(), req)
		require.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrEmptyStreet)
	})

	t.Run("Street too long is rejected", func(t *testing.T) {
		req := validCreateAddressRequest(newTestCustomer(t, customerSvc))
		req.Street = strings.Repeat("a", 201)
		_, err := svc.CreateAddress(t.Context(), req)
		require.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrStreetTooLong)
	})

	t.Run("Empty city is rejected", func(t *testing.T) {
		req := validCreateAddressRequest(newTestCustomer(t, customerSvc))
		req.City = "   "
		_, err := svc.CreateAddress(t.Context(), req)
		require.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrEmptyCity)
	})

	t.Run("City too long is rejected", func(t *testing.T) {
		req := validCreateAddressRequest(newTestCustomer(t, customerSvc))
		req.City = strings.Repeat("a", 101)
		_, err := svc.CreateAddress(t.Context(), req)
		require.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrCityTooLong)
	})

	t.Run("Empty state is rejected", func(t *testing.T) {
		req := validCreateAddressRequest(newTestCustomer(t, customerSvc))
		req.State = "   "
		_, err := svc.CreateAddress(t.Context(), req)
		require.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrEmptyState)
	})

	t.Run("State too long is rejected", func(t *testing.T) {
		req := validCreateAddressRequest(newTestCustomer(t, customerSvc))
		req.State = strings.Repeat("a", 101)
		_, err := svc.CreateAddress(t.Context(), req)
		require.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrStateTooLong)
	})

	t.Run("Empty zip is rejected", func(t *testing.T) {
		req := validCreateAddressRequest(newTestCustomer(t, customerSvc))
		req.Zip = "   "
		_, err := svc.CreateAddress(t.Context(), req)
		require.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrEmptyZip)
	})

	t.Run("Zip too long is rejected", func(t *testing.T) {
		req := validCreateAddressRequest(newTestCustomer(t, customerSvc))
		req.Zip = strings.Repeat("a", 21)
		_, err := svc.CreateAddress(t.Context(), req)
		require.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrZipTooLong)
	})

	t.Run("Empty country is rejected", func(t *testing.T) {
		req := validCreateAddressRequest(newTestCustomer(t, customerSvc))
		req.Country = "   "
		_, err := svc.CreateAddress(t.Context(), req)
		require.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrEmptyCountry)
	})

	t.Run("Country too long is rejected", func(t *testing.T) {
		req := validCreateAddressRequest(newTestCustomer(t, customerSvc))
		req.Country = strings.Repeat("a", 101)
		_, err := svc.CreateAddress(t.Context(), req)
		require.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrCountryTooLong)
	})

	t.Run("Phone too long is rejected", func(t *testing.T) {
		req := validCreateAddressRequest(newTestCustomer(t, customerSvc))
		req.Phone = strings.Repeat("a", 31)
		_, err := svc.CreateAddress(t.Context(), req)
		require.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrPhoneTooLong)
	})

	t.Run("Address got created with all fields", func(t *testing.T) {
		customerID := newTestCustomer(t, customerSvc)
		req := addressapp.CreateAddressRequest{
			CustomerID: customerID,
			Label:      "Home",
			FullName:   "John Doe",
			Street:     "123 Main St",
			City:       "New York",
			State:      "NY",
			Zip:        "10001",
			Country:    "USA",
			Phone:      "+1-555-1234",
			IsDefault:  false,
		}

		created := createTestAddress(t, svc, req)
		stored := getTestAddress(t, svc, created.ID, customerID)

		assert.NotEmpty(t, stored.ID)
		assert.Equal(t, customerID, stored.CustomerID)
		assert.Equal(t, "Home", stored.Label)
		assert.Equal(t, "John Doe", stored.FullName)
		assert.Equal(t, "123 Main St", stored.Street)
		assert.Equal(t, "New York", stored.City)
		assert.Equal(t, "NY", stored.State)
		assert.Equal(t, "10001", stored.Zip)
		assert.Equal(t, "USA", stored.Country)
		assert.Equal(t, "+1-555-1234", stored.Phone)
		assert.False(t, stored.IsDefault)
		assert.Equal(t, 1, stored.Version)
		assert.False(t, stored.CreatedAt.IsZero())
	})

	t.Run("Address without phone gets created", func(t *testing.T) {
		customerID := newTestCustomer(t, customerSvc)
		req := validCreateAddressRequest(customerID)
		req.Phone = ""

		created := createTestAddress(t, svc, req)
		stored := getTestAddress(t, svc, created.ID, customerID)

		assert.NotEmpty(t, stored.ID)
		assert.Equal(t, customerID, stored.CustomerID)
		assert.Equal(t, "Home", stored.Label)
		assert.Empty(t, stored.Phone)
		assert.Equal(t, 1, stored.Version)
		assert.False(t, stored.CreatedAt.IsZero())
	})

	t.Run("Address created as default", func(t *testing.T) {
		customerID := newTestCustomer(t, customerSvc)
		req := validCreateAddressRequest(customerID)
		req.IsDefault = true

		created := createTestAddress(t, svc, req)
		stored := getTestAddress(t, svc, created.ID, customerID)

		assert.NotEmpty(t, stored.ID)
		assert.Equal(t, customerID, stored.CustomerID)
		assert.Equal(t, "Home", stored.Label)
		assert.True(t, stored.IsDefault)
		assert.Equal(t, 1, stored.Version)
		assert.False(t, stored.CreatedAt.IsZero())
	})

	t.Run("Fields are trimmed on creation", func(t *testing.T) {
		customerID := newTestCustomer(t, customerSvc)
		req := addressapp.CreateAddressRequest{
			CustomerID: customerID,
			Label:      "  Home  ",
			FullName:   "  John Doe  ",
			Street:     "  123 Main St  ",
			City:       "  New York  ",
			State:      "  NY  ",
			Zip:        "  10001  ",
			Country:    "  USA  ",
			Phone:      "  +1-555-1234  ",
			IsDefault:  false,
		}

		created := createTestAddress(t, svc, req)
		stored := getTestAddress(t, svc, created.ID, customerID)

		assert.Equal(t, "Home", stored.Label)
		assert.Equal(t, "John Doe", stored.FullName)
		assert.Equal(t, "123 Main St", stored.Street)
		assert.Equal(t, "New York", stored.City)
		assert.Equal(t, "NY", stored.State)
		assert.Equal(t, "10001", stored.Zip)
		assert.Equal(t, "USA", stored.Country)
		assert.Equal(t, "+1-555-1234", stored.Phone)
	})

	t.Run("Country is normalized on creation", func(t *testing.T) {
		customerID := newTestCustomer(t, customerSvc)
		req := validCreateAddressRequest(customerID)
		req.Country = "brasil"

		created := createTestAddress(t, svc, req)
		stored := getTestAddress(t, svc, created.ID, customerID)

		assert.Equal(t, "BR", stored.Country)
	})
}
