package event

import (
	"context"
	addressclient "minimart/internal/address/presentation/module/client"
	catalogclient "minimart/internal/catalog/presentation/module/client"
	categoryclient "minimart/internal/category/presentation/module/client"
	customerclient "minimart/internal/customer/presentation/module/client"
	"minimart/internal/order/application"
	"minimart/internal/order/domain"
	"minimart/internal/order/infrastructure/memory"
	orderclient "minimart/internal/order/presentation/module/client"
	shippingclient "minimart/internal/shipping/presentation/module/client"
	"minimart/pkg/idempotency"
	"minimart/pkg/metrics"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

type stubModules struct {
	shippingclient.ShippingClient
	addressclient.AddressClient
}

func (stubModules) GetProduct(context.Context, string) (catalogclient.Product, error) {
	return catalogclient.Product{ID: "p1", Name: "Widget", PriceCents: 500}, nil
}

func (stubModules) GetCategory(context.Context, string) (categoryclient.CategoryDTO, error) {
	return categoryclient.CategoryDTO{ID: "cat-1", Name: "Test Category"}, nil
}

func (stubModules) ReserveStock(context.Context, string, []catalogclient.LineItem) error { return nil }

func (stubModules) ReleaseStock(context.Context, string) error { return nil }

func (stubModules) EnsureCustomer(context.Context, string) (customerclient.Customer, error) {
	return customerclient.Customer{ID: "cust-u1", UserID: "u1"}, nil
}

func (stubModules) GetCustomerByID(context.Context, string) (customerclient.Customer, error) {
	return customerclient.Customer{ID: "cust-u1", UserID: "u1"}, nil
}

func (stubModules) GetCustomerByUserID(context.Context, string) (customerclient.Customer, error) {
	return customerclient.Customer{ID: "cust-u1", UserID: "u1"}, nil
}

func (stubModules) PlaceOrder(_ context.Context, req orderclient.PlaceOrderRequest) (orderclient.Order, error) {
	return orderclient.Order{}, nil
}

func (stubModules) GetOrder(_ context.Context, id string) (orderclient.Order, error) {
	return orderclient.Order{}, nil
}

type noopIdem struct{}

func (noopIdem) Get(context.Context, string) (idempotency.Record, bool, error) {
	return idempotency.Record{}, false, nil
}

func (noopIdem) Reserve(context.Context, string, string, time.Duration) (bool, error) {
	return true, nil
}

func (noopIdem) Complete(context.Context, string, string, string, time.Duration) error { return nil }

func (noopIdem) Release(context.Context, string, string) error { return nil }

func seededHandler(t *testing.T) (*Handler, *memory.Repository) {
	t.Helper()
	repo := memory.New()
	order, err := domain.NewOrder("o1", "u1", "cust-u1", "k1", []domain.Item{
		{ProductID: "p1", Name: "Widget", UnitPriceCents: 500, Quantity: 1},
	})
	require.NoError(t, err)
	require.NoError(t, repo.Save(t.Context(), order))
	svc := application.NewService(repo, stubModules{}, noopIdem{}, metrics.Noop{}, time.Hour)
	return NewHandler(svc), repo
}
