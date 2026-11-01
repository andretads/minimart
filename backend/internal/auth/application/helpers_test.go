package application

import (
	"context"
	"errors"
	"minimart/internal/auth/infrastructure/memory"
	"minimart/internal/customer/presentation/module/client"
	"minimart/pkg/metrics"
	"testing"
)

type fakeProvisioner struct {
	err error
}

func (f *fakeProvisioner) EnsureCustomer(ctx context.Context, userID string) (client.Customer, error) {
	if f.err != nil {
		return client.Customer{}, f.err
	}
	return client.Customer{ID: "cust-" + userID, UserID: userID}, nil
}

type failingProvisioner struct{}

func (f *failingProvisioner) EnsureCustomer(ctx context.Context, userID string) (client.Customer, error) {
	return client.Customer{}, errors.New("provisioner unavailable")
}

func newTestService(t *testing.T, prov CustomerProvisioner) (*Service, *memory.Repository) {
	t.Helper()
	repo := memory.New()
	svc := NewService(repo, prov, metrics.Noop{})
	return svc, repo
}
