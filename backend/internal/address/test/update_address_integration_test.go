//go:build integration

package test

import (
	"minimart/internal/address/domain"
	"strings"
	"testing"
	"time"

	addressapp "minimart/internal/address/application"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUpdateAddressIntegration(t *testing.T) {
	svc, customerSvc := newAddressServices(t)

	t.Run("Empty customer ID is rejected", func(t *testing.T) {
		customerID := newTestCustomer(t, customerSvc)
		addr := createTestAddress(t, svc, validCreateAddressRequest(customerID))

		req := validUpdateAddressRequest(addr.ID, "", addr.Version)
		req.Label = "Work"
		_, err := svc.UpdateAddress(t.Context(), req)
		require.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrEmptyCustomerID)
	})

	t.Run("Invalid version is rejected", func(t *testing.T) {
		customerID := newTestCustomer(t, customerSvc)
		addr := createTestAddress(t, svc, validCreateAddressRequest(customerID))

		req := validUpdateAddressRequest(addr.ID, customerID, 0)
		req.Label = "Work"
		_, err := svc.UpdateAddress(t.Context(), req)
		require.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrInvalidVersion)
	})

	t.Run("Unknown ID is rejected", func(t *testing.T) {
		customerID := newTestCustomer(t, customerSvc)

		req := validUpdateAddressRequest("missing", customerID, 1)
		req.Label = "Work"
		_, err := svc.UpdateAddress(t.Context(), req)
		require.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrAddressNotFound)
	})

	t.Run("Address owned by another customer is not found", func(t *testing.T) {
		ownerID := newTestCustomer(t, customerSvc)
		otherID := newTestCustomer(t, customerSvc)
		addr := createTestAddress(t, svc, validCreateAddressRequest(ownerID))

		req := validUpdateAddressRequest(addr.ID, otherID, 1)
		req.Label = "Work"
		_, err := svc.UpdateAddress(t.Context(), req)
		require.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrAddressNotFound)
	})

	t.Run("Stale version is rejected", func(t *testing.T) {
		customerID := newTestCustomer(t, customerSvc)
		addr := createTestAddress(t, svc, validCreateAddressRequest(customerID))

		currentVersion := addr.Version

		firstReq := validUpdateAddressRequest(addr.ID, customerID, currentVersion)
		firstReq.Label = "First"
		_, err := svc.UpdateAddress(t.Context(), firstReq)
		require.NoError(t, err)

		staleReq := validUpdateAddressRequest(addr.ID, customerID, currentVersion)
		staleReq.Label = "Work"
		_, err = svc.UpdateAddress(t.Context(), staleReq)
		require.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrInvalidVersion)
	})

	t.Run("Empty label is rejected", func(t *testing.T) {
		customerID := newTestCustomer(t, customerSvc)
		addr := createTestAddress(t, svc, validCreateAddressRequest(customerID))

		req := validUpdateAddressRequest(addr.ID, customerID, addr.Version)
		req.Label = "   "
		_, err := svc.UpdateAddress(t.Context(), req)
		require.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrEmptyLabel)
	})

	t.Run("Label too long is rejected", func(t *testing.T) {
		customerID := newTestCustomer(t, customerSvc)
		addr := createTestAddress(t, svc, validCreateAddressRequest(customerID))

		req := validUpdateAddressRequest(addr.ID, customerID, addr.Version)
		req.Label = strings.Repeat("a", 51)
		_, err := svc.UpdateAddress(t.Context(), req)
		require.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrLabelTooLong)
	})

	t.Run("Empty full name is rejected", func(t *testing.T) {
		customerID := newTestCustomer(t, customerSvc)
		addr := createTestAddress(t, svc, validCreateAddressRequest(customerID))

		req := validUpdateAddressRequest(addr.ID, customerID, addr.Version)
		req.FullName = "   "
		_, err := svc.UpdateAddress(t.Context(), req)
		require.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrEmptyFullName)
	})

	t.Run("Full name too long is rejected", func(t *testing.T) {
		customerID := newTestCustomer(t, customerSvc)
		addr := createTestAddress(t, svc, validCreateAddressRequest(customerID))

		req := validUpdateAddressRequest(addr.ID, customerID, addr.Version)
		req.FullName = strings.Repeat("a", 101)
		_, err := svc.UpdateAddress(t.Context(), req)
		require.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrFullNameTooLong)
	})

	t.Run("Empty street is rejected", func(t *testing.T) {
		customerID := newTestCustomer(t, customerSvc)
		addr := createTestAddress(t, svc, validCreateAddressRequest(customerID))

		req := validUpdateAddressRequest(addr.ID, customerID, addr.Version)
		req.Street = "   "
		_, err := svc.UpdateAddress(t.Context(), req)
		require.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrEmptyStreet)
	})

	t.Run("Street too long is rejected", func(t *testing.T) {
		customerID := newTestCustomer(t, customerSvc)
		addr := createTestAddress(t, svc, validCreateAddressRequest(customerID))

		req := validUpdateAddressRequest(addr.ID, customerID, addr.Version)
		req.Street = strings.Repeat("a", 201)
		_, err := svc.UpdateAddress(t.Context(), req)
		require.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrStreetTooLong)
	})

	t.Run("Empty city is rejected", func(t *testing.T) {
		customerID := newTestCustomer(t, customerSvc)
		addr := createTestAddress(t, svc, validCreateAddressRequest(customerID))

		req := validUpdateAddressRequest(addr.ID, customerID, addr.Version)
		req.City = "   "
		_, err := svc.UpdateAddress(t.Context(), req)
		require.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrEmptyCity)
	})

	t.Run("City too long is rejected", func(t *testing.T) {
		customerID := newTestCustomer(t, customerSvc)
		addr := createTestAddress(t, svc, validCreateAddressRequest(customerID))

		req := validUpdateAddressRequest(addr.ID, customerID, addr.Version)
		req.City = strings.Repeat("a", 101)
		_, err := svc.UpdateAddress(t.Context(), req)
		require.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrCityTooLong)
	})

	t.Run("Empty state is rejected", func(t *testing.T) {
		customerID := newTestCustomer(t, customerSvc)
		addr := createTestAddress(t, svc, validCreateAddressRequest(customerID))

		req := validUpdateAddressRequest(addr.ID, customerID, addr.Version)
		req.State = "   "
		_, err := svc.UpdateAddress(t.Context(), req)
		require.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrEmptyState)
	})

	t.Run("State too long is rejected", func(t *testing.T) {
		customerID := newTestCustomer(t, customerSvc)
		addr := createTestAddress(t, svc, validCreateAddressRequest(customerID))

		req := validUpdateAddressRequest(addr.ID, customerID, addr.Version)
		req.State = strings.Repeat("a", 101)
		_, err := svc.UpdateAddress(t.Context(), req)
		require.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrStateTooLong)
	})

	t.Run("Empty zip is rejected", func(t *testing.T) {
		customerID := newTestCustomer(t, customerSvc)
		addr := createTestAddress(t, svc, validCreateAddressRequest(customerID))

		req := validUpdateAddressRequest(addr.ID, customerID, addr.Version)
		req.Zip = "   "
		_, err := svc.UpdateAddress(t.Context(), req)
		require.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrEmptyZip)
	})

	t.Run("Zip too long is rejected", func(t *testing.T) {
		customerID := newTestCustomer(t, customerSvc)
		addr := createTestAddress(t, svc, validCreateAddressRequest(customerID))

		req := validUpdateAddressRequest(addr.ID, customerID, addr.Version)
		req.Zip = strings.Repeat("a", 21)
		_, err := svc.UpdateAddress(t.Context(), req)
		require.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrZipTooLong)
	})

	t.Run("Empty country is rejected", func(t *testing.T) {
		customerID := newTestCustomer(t, customerSvc)
		addr := createTestAddress(t, svc, validCreateAddressRequest(customerID))

		req := validUpdateAddressRequest(addr.ID, customerID, addr.Version)
		req.Country = "   "
		_, err := svc.UpdateAddress(t.Context(), req)
		require.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrEmptyCountry)
	})

	t.Run("Country too long is rejected", func(t *testing.T) {
		customerID := newTestCustomer(t, customerSvc)
		addr := createTestAddress(t, svc, validCreateAddressRequest(customerID))

		req := validUpdateAddressRequest(addr.ID, customerID, addr.Version)
		req.Country = strings.Repeat("a", 101)
		_, err := svc.UpdateAddress(t.Context(), req)
		require.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrCountryTooLong)
	})

	t.Run("Phone too long is rejected", func(t *testing.T) {
		customerID := newTestCustomer(t, customerSvc)
		addr := createTestAddress(t, svc, validCreateAddressRequest(customerID))

		req := validUpdateAddressRequest(addr.ID, customerID, addr.Version)
		req.Phone = strings.Repeat("a", 31)
		_, err := svc.UpdateAddress(t.Context(), req)
		require.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrPhoneTooLong)
	})

	t.Run("Address got updated with no problems", func(t *testing.T) {
		customerID := newTestCustomer(t, customerSvc)
		req := validCreateAddressRequest(customerID)
		req.Label = "Home"
		addr := createTestAddress(t, svc, req)

		updateReq := addressapp.UpdateAddressRequest{
			ID:         addr.ID,
			CustomerID: customerID,
			Label:      "Work",
			FullName:   "Jane Smith",
			Street:     "456 Office Ave",
			City:       "San Francisco",
			State:      "CA",
			Zip:        "94102",
			Country:    "USA",
			Phone:      "+1-555-5678",
			Version:    addr.Version,
		}
		_, err := svc.UpdateAddress(t.Context(), updateReq)
		require.NoError(t, err)

		stored := getTestAddress(t, svc, addr.ID, customerID)
		assert.Equal(t, "Work", stored.Label)
		assert.Equal(t, "Jane Smith", stored.FullName)
		assert.Equal(t, "456 Office Ave", stored.Street)
		assert.Equal(t, "San Francisco", stored.City)
		assert.Equal(t, "CA", stored.State)
		assert.Equal(t, "94102", stored.Zip)
		assert.Equal(t, "USA", stored.Country)
		assert.Equal(t, "+1-555-5678", stored.Phone)
		assert.Equal(t, 2, stored.Version)
	})

	t.Run("Version is bumped on successful update", func(t *testing.T) {
		customerID := newTestCustomer(t, customerSvc)
		addr := createTestAddress(t, svc, validCreateAddressRequest(customerID))

		req := validUpdateAddressRequest(addr.ID, customerID, addr.Version)
		req.Label = "Work"
		_, err := svc.UpdateAddress(t.Context(), req)
		require.NoError(t, err)

		stored := getTestAddress(t, svc, addr.ID, customerID)
		assert.Equal(t, 2, stored.Version)
	})

	t.Run("Created_at is unchanged", func(t *testing.T) {
		customerID := newTestCustomer(t, customerSvc)
		addr := createTestAddress(t, svc, validCreateAddressRequest(customerID))

		req := validUpdateAddressRequest(addr.ID, customerID, addr.Version)
		req.Label = "Work"
		_, err := svc.UpdateAddress(t.Context(), req)
		require.NoError(t, err)

		stored := getTestAddress(t, svc, addr.ID, customerID)
		assert.WithinDuration(t, addr.CreatedAt, stored.CreatedAt, time.Second)
	})

	t.Run("Fields are trimmed on update", func(t *testing.T) {
		customerID := newTestCustomer(t, customerSvc)
		addr := createTestAddress(t, svc, validCreateAddressRequest(customerID))

		req := validUpdateAddressRequest(addr.ID, customerID, addr.Version)
		req.Label = "  Work  "
		_, err := svc.UpdateAddress(t.Context(), req)
		require.NoError(t, err)

		stored := getTestAddress(t, svc, addr.ID, customerID)
		assert.Equal(t, "Work", stored.Label)
	})

	t.Run("Setting is_default promotes the address to default", func(t *testing.T) {
		customerID := newTestCustomer(t, customerSvc)
		req := validCreateAddressRequest(customerID)
		req.IsDefault = false
		addr := createTestAddress(t, svc, req)

		updateReq := validUpdateAddressRequest(addr.ID, customerID, addr.Version)
		updateReq.Label = "Work"
		updateReq.IsDefault = true
		_, err := svc.UpdateAddress(t.Context(), updateReq)
		require.NoError(t, err)

		stored := getTestAddress(t, svc, addr.ID, customerID)
		assert.Equal(t, addr.ID, stored.ID)
		assert.True(t, stored.IsDefault)
		// The feature expects version 2, but the use case persists the detail
		// update (v2) and then promotes to default (v3).
		assert.Equal(t, 3, stored.Version)
	})
}
