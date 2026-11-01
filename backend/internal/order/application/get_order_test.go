package application

import (
	"context"
	addressclient "minimart/internal/address/presentation/module/client"
	catalogclient "minimart/internal/catalog/presentation/module/client"
	categoryclient "minimart/internal/category/presentation/module/client"
	customerclient "minimart/internal/customer/presentation/module/client"
	"minimart/internal/order/domain"
	"minimart/internal/order/infrastructure/memory"
	orderclient "minimart/internal/order/presentation/module/client"
	shippingclient "minimart/internal/shipping/presentation/module/client"
	"minimart/pkg/idempotency"
	"minimart/pkg/metrics"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// minimal stub to exercise GetOrder (doesn't need full catalog/customer setup).
type stubModules struct {
	shippingclient.ShippingClient
	addressclient.AddressClient
}

func (stubModules) GetProduct(context.Context, string) (catalogclient.Product, error) {
	return catalogclient.Product{}, nil
}

func (stubModules) GetCategory(context.Context, string) (categoryclient.CategoryDTO, error) {
	return categoryclient.CategoryDTO{}, nil
}

func (stubModules) ReserveStock(context.Context, string, []catalogclient.LineItem) error { return nil }

func (stubModules) ReleaseStock(context.Context, string) error { return nil }

func (stubModules) EnsureCustomer(_ context.Context, userID string) (customerclient.Customer, error) {
	return customerclient.Customer{ID: "cust-" + userID, UserID: userID}, nil
}

func (stubModules) GetCustomerByID(_ context.Context, id string) (customerclient.Customer, error) {
	return customerclient.Customer{ID: id, UserID: "user-" + id}, nil
}

func (stubModules) GetCustomerByUserID(_ context.Context, userID string) (customerclient.Customer, error) {
	return customerclient.Customer{ID: "cust-" + userID, UserID: userID}, nil
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

func TestGetOrder(t *testing.T) {
	repo := memory.New()
	modules := stubModules{}
	svc := NewService(repo, modules, noopIdem{}, metrics.Noop{}, time.Hour)

	placed, err := svc.PlaceOrder(t.Context(), PlaceOrderRequest{
		UserID:         "u1",
		IdempotencyKey: "k1",
		Items:          []PlaceOrderItem{{ProductID: "p1", Quantity: 1}},
	})

	require.NoError(t, err)

	t.Run("returns the stored order", func(t *testing.T) {
		got, err := svc.GetOrder(t.Context(), placed.ID)
		require.NoError(t, err)
		assert.Equal(t, placed.ID, got.ID)
		assert.Equal(t, "u1", got.UserID)
	})

	t.Run("unknown id reports not found", func(t *testing.T) {
		_, err := svc.GetOrder(t.Context(), "missing")
		assert.ErrorIs(t, err, domain.ErrOrderNotFound)
	})
}
