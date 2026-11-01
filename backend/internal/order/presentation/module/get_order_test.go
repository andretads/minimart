package module

import (
	"context"
	"errors"
	catalogclient "minimart/internal/catalog/presentation/module/client"
	customerclient "minimart/internal/customer/presentation/module/client"
	"minimart/internal/order/application"
	"minimart/internal/order/domain"
	"minimart/internal/order/infrastructure/memory"
	orderclient "minimart/internal/order/presentation/module/client"
	shippingclient "minimart/internal/shipping/presentation/module/client"
	"minimart/pkg/idempotency"
	"minimart/pkg/metrics"
	"minimart/pkg/module/contracts"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fakeCatalogForGetOrder implements the catalogclient.Catalog interface for testing.
type fakeCatalogForGetOrder struct{}

func (f *fakeCatalogForGetOrder) GetProduct(ctx context.Context, id string) (catalogclient.Product, error) {
	return catalogclient.Product{}, nil
}

func (f *fakeCatalogForGetOrder) ReserveStock(ctx context.Context, orderID string, items []catalogclient.LineItem) error {
	return nil
}

func (f *fakeCatalogForGetOrder) ReleaseStock(ctx context.Context, orderID string) error { return nil }

// fakeCustomerForGetOrder implements the customerclient.CustomerClient interface for testing.
type fakeCustomerForGetOrder struct{}

func (f *fakeCustomerForGetOrder) EnsureCustomer(ctx context.Context, userID string) (customerclient.Customer, error) {
	return customerclient.Customer{}, nil
}

func (f *fakeCustomerForGetOrder) GetCustomerByID(ctx context.Context, id string) (customerclient.Customer, error) {
	return customerclient.Customer{}, nil
}

func (f *fakeCustomerForGetOrder) GetCustomerByUserID(ctx context.Context, userID string) (customerclient.Customer, error) {
	return customerclient.Customer{}, nil
}

// fakeShippingForGetOrder implements the shippingclient.ShippingClient interface for testing.
type fakeShippingForGetOrder struct{}

func (f *fakeShippingForGetOrder) ListMethods(ctx context.Context) ([]shippingclient.ShippingMethod, error) {
	return nil, nil
}

func (f *fakeShippingForGetOrder) GetMethod(ctx context.Context, id string) (shippingclient.ShippingMethod, error) {
	return shippingclient.ShippingMethod{}, nil
}

func (f *fakeShippingForGetOrder) CreateMethod(ctx context.Context, req shippingclient.CreateMethodRequest) (shippingclient.ShippingMethod, error) {
	return shippingclient.ShippingMethod{}, nil
}

func (f *fakeShippingForGetOrder) UpdateMethod(ctx context.Context, id string, req shippingclient.UpdateMethodRequest) (shippingclient.ShippingMethod, error) {
	return shippingclient.ShippingMethod{}, nil
}

func (f *fakeShippingForGetOrder) ListZones(ctx context.Context) ([]shippingclient.ShippingZone, error) {
	return nil, nil
}

func (f *fakeShippingForGetOrder) GetZone(ctx context.Context, id string) (shippingclient.ShippingZone, error) {
	return shippingclient.ShippingZone{}, nil
}

func (f *fakeShippingForGetOrder) CreateZone(ctx context.Context, req shippingclient.CreateZoneRequest) (shippingclient.ShippingZone, error) {
	return shippingclient.ShippingZone{}, nil
}

func (f *fakeShippingForGetOrder) UpdateZone(ctx context.Context, id string, req shippingclient.UpdateZoneRequest) (shippingclient.ShippingZone, error) {
	return shippingclient.ShippingZone{}, nil
}

func (f *fakeShippingForGetOrder) ListRates(ctx context.Context, zoneID, methodID string) ([]shippingclient.ShippingRate, error) {
	return nil, nil
}

func (f *fakeShippingForGetOrder) GetRate(ctx context.Context, id string) (shippingclient.ShippingRate, error) {
	return shippingclient.ShippingRate{}, nil
}

func (f *fakeShippingForGetOrder) CreateRate(ctx context.Context, req shippingclient.CreateRateRequest) (shippingclient.ShippingRate, error) {
	return shippingclient.ShippingRate{}, nil
}

func (f *fakeShippingForGetOrder) UpdateRate(ctx context.Context, id string, req shippingclient.UpdateRateRequest) (shippingclient.ShippingRate, error) {
	return shippingclient.ShippingRate{}, nil
}

func (f *fakeShippingForGetOrder) GetQuote(ctx context.Context, req shippingclient.GetQuoteRequest) (shippingclient.ShippingQuote, error) {
	return shippingclient.ShippingQuote{}, nil
}

func (f *fakeShippingForGetOrder) CreateQuote(ctx context.Context, req shippingclient.CreateQuoteRequest) (shippingclient.ShippingQuote, error) {
	return shippingclient.ShippingQuote{}, nil
}

func (f *fakeShippingForGetOrder) ValidateQuote(ctx context.Context, req shippingclient.ValidateQuoteRequest) (shippingclient.ShippingQuote, error) {
	return shippingclient.ShippingQuote{}, nil
}

func (f *fakeShippingForGetOrder) PriceQuote(ctx context.Context, req shippingclient.PriceQuoteRequest) (shippingclient.ShippingQuote, error) {
	return shippingclient.ShippingQuote{}, nil
}

func (f *fakeShippingForGetOrder) GetShipment(ctx context.Context, id string) (shippingclient.Shipment, error) {
	return shippingclient.Shipment{}, nil
}

func (f *fakeShippingForGetOrder) GetShipmentByOrderID(ctx context.Context, orderID string) (shippingclient.Shipment, error) {
	return shippingclient.Shipment{}, nil
}

func (f *fakeShippingForGetOrder) ListShipments(ctx context.Context, userID string, page, pageSize int) ([]shippingclient.Shipment, int, error) {
	return nil, 0, nil
}

func (f *fakeShippingForGetOrder) GetTrackingEvents(ctx context.Context, shipmentID string) ([]shippingclient.TrackingEvent, error) {
	return nil, nil
}

// fakeIdempotencyForGetOrder implements idempotency.Store for testing.
type fakeIdempotencyForGetOrder struct{}

func (f *fakeIdempotencyForGetOrder) Get(ctx context.Context, key string) (idempotency.Record, bool, error) {
	return idempotency.Record{}, false, nil
}

func (f *fakeIdempotencyForGetOrder) Reserve(ctx context.Context, key, fp string, ttl time.Duration) (bool, error) {
	return true, nil
}

func (f *fakeIdempotencyForGetOrder) Complete(ctx context.Context, key, fp, resourceID string, ttl time.Duration) error {
	return nil
}

func (f *fakeIdempotencyForGetOrder) Release(ctx context.Context, key, _ string) error { return nil }

// failingRepoForGetOrder makes GetByID fail with a non-not-found error so the
// pass-through branch can be exercised.
type failingRepoForGetOrder struct {
	err error
}

func (f failingRepoForGetOrder) Save(context.Context, domain.Order) error { return nil }
func (f failingRepoForGetOrder) GetByID(context.Context, string) (domain.Order, error) {
	return domain.Order{}, f.err
}

func (f failingRepoForGetOrder) GetByIdempotencyKey(context.Context, string) (domain.Order, error) {
	return domain.Order{}, domain.ErrOrderNotFound
}

func (f failingRepoForGetOrder) Update(context.Context, domain.Order, domain.Status) error {
	return nil
}

func (f failingRepoForGetOrder) ListByUserID(context.Context, string, int, int) ([]domain.Order, error) {
	return nil, nil
}

func (f failingRepoForGetOrder) CountByUserID(context.Context, string) (int, error) { return 0, nil }

func (f failingRepoForGetOrder) ListPlacedBefore(context.Context, time.Time, int) ([]domain.Order, error) {
	return nil, nil
}

func TestOrderGetOrder(t *testing.T) {
	catalog := &fakeCatalogForGetOrder{}
	customer := &fakeCustomerForGetOrder{}
	shipping := &fakeShippingForGetOrder{}
	idem := &fakeIdempotencyForGetOrder{}
	repo := memory.New()

	ctr := &contracts.Contracts{
		CatalogClient:  catalog,
		CustomerClient: customer,
		ShippingClient: shipping,
	}
	_ = ctr.Verify()

	svc := application.NewService(repo, ctr, idem, metrics.Noop{}, time.Hour)
	adapter := New(svc)

	t.Run("returns the order from the service", func(t *testing.T) {
		ord, err := domain.NewOrder(
			"order-1", "user-1", "cust-1", "idempotency-key",
			[]domain.Item{{ProductID: "prod-1", Name: "Widget", UnitPriceCents: 500, Quantity: 1}},
		)
		require.NoError(t, err)
		require.NoError(t, repo.Save(t.Context(), ord))

		got, err := adapter.GetOrder(t.Context(), "order-1")
		require.NoError(t, err)
		assert.Equal(t, "order-1", got.ID)
		assert.Equal(t, "user-1", got.UserID)
		assert.Equal(t, "cust-1", got.CustomerID)
		assert.Equal(t, int64(500), got.TotalCents)
	})

	t.Run("translates not found into client sentinel", func(t *testing.T) {
		_, err := adapter.GetOrder(t.Context(), "missing")
		assert.ErrorIs(t, err, orderclient.ErrOrderNotFound)
	})

	t.Run("carries the shipping snapshot across the contract", func(t *testing.T) {
		ord, err := domain.NewOrder(
			"order-2", "user-1", "cust-1", "idempotency-key-2",
			[]domain.Item{{ProductID: "prod-1", Name: "Widget", UnitPriceCents: 500, Quantity: 1}},
		)
		require.NoError(t, err)
		ord.ShippingQuoteID = "quote-1"
		ord.ShippingMethodID = "method-1"
		ord.ShippingMethodName = "Standard"
		ord.ShippingCostCents = 250
		ord.DestinationSnapshot = domain.DestinationSnapshot{
			FullName: "Ada Lovelace",
			Street:   "1 Main St",
			City:     "Testville",
			Zip:      "12345",
			Country:  "BR",
		}
		require.NoError(t, repo.Save(t.Context(), ord))

		got, err := adapter.GetOrder(t.Context(), "order-2")
		require.NoError(t, err)
		assert.Equal(t, "quote-1", got.ShippingQuoteID)
		assert.Equal(t, "method-1", got.ShippingMethodID)
		assert.Equal(t, "Standard", got.ShippingMethodName)
		assert.Equal(t, int64(250), got.ShippingCostCents)
		assert.Equal(t, "Ada Lovelace", got.DestinationSnapshot.FullName)
		assert.Equal(t, "1 Main St", got.DestinationSnapshot.Street)
	})

	t.Run("passes through unexpected errors unchanged", func(t *testing.T) {
		boom := errors.New("boom")
		svc := application.NewService(failingRepoForGetOrder{err: boom}, ctr, idem, metrics.Noop{}, time.Hour)
		adapter := New(svc)

		_, err := adapter.GetOrder(t.Context(), "order-1")
		require.ErrorIs(t, err, boom)
		assert.NotErrorIs(t, err, orderclient.ErrOrderNotFound)
	})
}
