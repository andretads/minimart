package application

import (
	"minimart/internal/address/domain"
	"minimart/internal/address/infrastructure/memory"
	"minimart/pkg/metrics"
	"testing"

	"github.com/stretchr/testify/require"
)

func newTestService(t *testing.T) (*Service, *memory.Repository) {
	t.Helper()
	repo := memory.New()
	return NewService(repo, metrics.Noop{}), repo
}

func validCreateAddressRequest(customerID string) CreateAddressRequest {
	return CreateAddressRequest{
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
}

func validUpdateAddressRequest(id, customerID string, version int) UpdateAddressRequest {
	return UpdateAddressRequest{
		ID:         id,
		CustomerID: customerID,
		Label:      "Home",
		FullName:   "John Doe",
		Street:     "123 Main St",
		City:       "New York",
		State:      "NY",
		Zip:        "10001",
		Country:    "USA",
		Phone:      "+1-555-1234",
		Version:    version,
	}
}

func createTestAddress(t *testing.T, svc *Service, req CreateAddressRequest) domain.Address {
	t.Helper()
	a, err := svc.CreateAddress(t.Context(), req)
	require.NoError(t, err)
	return a
}
