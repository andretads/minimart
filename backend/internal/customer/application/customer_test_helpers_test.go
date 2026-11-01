package application

import (
	"minimart/internal/customer/domain"
	"minimart/internal/customer/infrastructure/memory"
	"minimart/pkg/metrics"
	"testing"

	"github.com/stretchr/testify/require"
)

func newTestService(t *testing.T) (*Service, *memory.Repository) {
	t.Helper()
	repo := memory.New()
	return NewService(repo, metrics.Noop{}), repo
}

func validCreateCustomerRequest() CreateCustomerRequest {
	return CreateCustomerRequest{
		UserID:      "user-1",
		DisplayName: "John",
	}
}

func createTestCustomer(t *testing.T, svc *Service, req CreateCustomerRequest) domain.Customer {
	t.Helper()
	c, err := svc.CreateCustomer(t.Context(), req)
	require.NoError(t, err)
	return c
}
