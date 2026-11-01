package module

import (
	"context"
	"errors"
	catalogclient "minimart/internal/catalog/presentation/module/client"
	customerclient "minimart/internal/customer/presentation/module/client"
	"minimart/internal/order/application"
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

// fakeCatalogForPlaceOrder implements the catalogclient.Catalog interface for testing.
type fakeCatalogForPlaceOrder struct {
	products map[string]catalogclient.Product
	err      error
}

func (f *fakeCatalogForPlaceOrder) GetProduct(ctx context.Context, id string) (catalogclient.Product, error) {
	if f.err != nil {
		return catalogclient.Product{}, f.err
	}
	if p, ok := f.products[id]; ok {
		return p, nil
	}
	return catalogclient.Product{}, catalogclient.ErrProductNotFound
}

func (f *fakeCatalogForPlaceOrder) ReserveStock(ctx context.Context, orderID string, items []catalogclient.LineItem) error {
	if f.err != nil {
		return f.err
	}
	for _, item := range items {
		if _, ok := f.products[item.ProductID]; !ok {
			return catalogclient.ErrProductNotFound
		}
	}
	return nil
}

func (f *fakeCatalogForPlaceOrder) ReleaseStock(ctx context.Context, orderID string) error {
	return nil
}

// fakeCustomerForPlaceOrder implements the customerclient.CustomerClient interface for testing.
type fakeCustomerForPlaceOrder struct {
	customers map[string]customerclient.Customer
	err       error
}

func (f *fakeCustomerForPlaceOrder) EnsureCustomer(ctx context.Context, userID string) (customerclient.Customer, error) {
	if f.err != nil {
		return customerclient.Customer{}, f.err
	}
	if c, ok := f.customers[userID]; ok {
		return c, nil
	}
	return customerclient.Customer{}, customerclient.ErrCustomerNotFound
}

func (f *fakeCustomerForPlaceOrder) GetCustomerByID(ctx context.Context, id string) (customerclient.Customer, error) {
	if f.err != nil {
		return customerclient.Customer{}, f.err
	}
	for _, c := range f.customers {
		if c.ID == id {
			return c, nil
		}
	}
	return customerclient.Customer{}, customerclient.ErrCustomerNotFound
}

func (f *fakeCustomerForPlaceOrder) GetCustomerByUserID(ctx context.Context, userID string) (customerclient.Customer, error) {
	if f.err != nil {
		return customerclient.Customer{}, f.err
	}
	if c, ok := f.customers[userID]; ok {
		return c, nil
	}
	return customerclient.Customer{}, customerclient.ErrCustomerNotFound
}

// fakeShippingForPlaceOrder implements the shippingclient.ShippingClient interface for testing.
type fakeShippingForPlaceOrder struct {
	quotes map[string]shippingclient.ShippingQuote
	err    error
}

func (f *fakeShippingForPlaceOrder) GetQuote(ctx context.Context, req shippingclient.GetQuoteRequest) (shippingclient.ShippingQuote, error) {
	if f.err != nil {
		return shippingclient.ShippingQuote{}, f.err
	}
	if q, ok := f.quotes[req.ID]; ok {
		return q, nil
	}
	return shippingclient.ShippingQuote{}, shippingclient.ErrQuoteNotFound
}

func (f *fakeShippingForPlaceOrder) CreateQuote(ctx context.Context, req shippingclient.CreateQuoteRequest) (shippingclient.ShippingQuote, error) {
	if f.err != nil {
		return shippingclient.ShippingQuote{}, f.err
	}
	return shippingclient.ShippingQuote{}, nil
}

func (f *fakeShippingForPlaceOrder) ValidateQuote(ctx context.Context, req shippingclient.ValidateQuoteRequest) (shippingclient.ShippingQuote, error) {
	if f.err != nil {
		return shippingclient.ShippingQuote{}, f.err
	}
	if q, ok := f.quotes[req.ID]; ok {
		return q, nil
	}
	return shippingclient.ShippingQuote{}, shippingclient.ErrQuoteNotFound
}

func (f *fakeShippingForPlaceOrder) PriceQuote(ctx context.Context, req shippingclient.PriceQuoteRequest) (shippingclient.ShippingQuote, error) {
	if f.err != nil {
		return shippingclient.ShippingQuote{}, f.err
	}
	if q, ok := f.quotes[req.ID]; ok {
		return q, nil
	}
	return shippingclient.ShippingQuote{}, shippingclient.ErrQuoteNotFound
}

// Implement other required methods as no-ops
func (f *fakeShippingForPlaceOrder) ListMethods(ctx context.Context) ([]shippingclient.ShippingMethod, error) {
	return nil, nil
}

func (f *fakeShippingForPlaceOrder) GetMethod(ctx context.Context, id string) (shippingclient.ShippingMethod, error) {
	return shippingclient.ShippingMethod{}, nil
}

func (f *fakeShippingForPlaceOrder) CreateMethod(ctx context.Context, req shippingclient.CreateMethodRequest) (shippingclient.ShippingMethod, error) {
	return shippingclient.ShippingMethod{}, nil
}

func (f *fakeShippingForPlaceOrder) UpdateMethod(ctx context.Context, id string, req shippingclient.UpdateMethodRequest) (shippingclient.ShippingMethod, error) {
	return shippingclient.ShippingMethod{}, nil
}

func (f *fakeShippingForPlaceOrder) ListZones(ctx context.Context) ([]shippingclient.ShippingZone, error) {
	return nil, nil
}

func (f *fakeShippingForPlaceOrder) GetZone(ctx context.Context, id string) (shippingclient.ShippingZone, error) {
	return shippingclient.ShippingZone{}, nil
}

func (f *fakeShippingForPlaceOrder) CreateZone(ctx context.Context, req shippingclient.CreateZoneRequest) (shippingclient.ShippingZone, error) {
	return shippingclient.ShippingZone{}, nil
}

func (f *fakeShippingForPlaceOrder) UpdateZone(ctx context.Context, id string, req shippingclient.UpdateZoneRequest) (shippingclient.ShippingZone, error) {
	return shippingclient.ShippingZone{}, nil
}

func (f *fakeShippingForPlaceOrder) ListRates(ctx context.Context, zoneID, methodID string) ([]shippingclient.ShippingRate, error) {
	return nil, nil
}

func (f *fakeShippingForPlaceOrder) GetRate(ctx context.Context, id string) (shippingclient.ShippingRate, error) {
	return shippingclient.ShippingRate{}, nil
}

func (f *fakeShippingForPlaceOrder) CreateRate(ctx context.Context, req shippingclient.CreateRateRequest) (shippingclient.ShippingRate, error) {
	return shippingclient.ShippingRate{}, nil
}

func (f *fakeShippingForPlaceOrder) UpdateRate(ctx context.Context, id string, req shippingclient.UpdateRateRequest) (shippingclient.ShippingRate, error) {
	return shippingclient.ShippingRate{}, nil
}

func (f *fakeShippingForPlaceOrder) GetShipment(ctx context.Context, id string) (shippingclient.Shipment, error) {
	return shippingclient.Shipment{}, nil
}

func (f *fakeShippingForPlaceOrder) GetShipmentByOrderID(ctx context.Context, orderID string) (shippingclient.Shipment, error) {
	return shippingclient.Shipment{}, nil
}

func (f *fakeShippingForPlaceOrder) ListShipments(ctx context.Context, userID string, page, pageSize int) ([]shippingclient.Shipment, int, error) {
	return nil, 0, nil
}

func (f *fakeShippingForPlaceOrder) GetTrackingEvents(ctx context.Context, shipmentID string) ([]shippingclient.TrackingEvent, error) {
	return nil, nil
}

// fakeIdempotencyForPlaceOrder implements idempotency.Store for testing.
type fakeIdempotencyForPlaceOrder struct {
	keys map[string]string
}

func (f *fakeIdempotencyForPlaceOrder) Get(ctx context.Context, key string) (idempotency.Record, bool, error) {
	if v, ok := f.keys[key]; ok {
		return idempotency.Record{Fingerprint: v}, true, nil
	}
	return idempotency.Record{}, false, nil
}

func (f *fakeIdempotencyForPlaceOrder) Reserve(ctx context.Context, key, fp string, ttl time.Duration) (bool, error) {
	if _, ok := f.keys[key]; ok {
		return false, nil
	}
	f.keys[key] = fp
	return true, nil
}

func (f *fakeIdempotencyForPlaceOrder) Complete(ctx context.Context, key, fp, resourceID string, ttl time.Duration) error {
	return nil
}

func (f *fakeIdempotencyForPlaceOrder) Release(ctx context.Context, key, _ string) error { return nil }

func TestOrderPlaceOrder(t *testing.T) {
	catalog := &fakeCatalogForPlaceOrder{
		products: map[string]catalogclient.Product{
			"prod-1": {ID: "prod-1", Name: "Widget", PriceCents: 500},
		},
	}
	customer := &fakeCustomerForPlaceOrder{
		customers: map[string]customerclient.Customer{
			"user-1": {ID: "cust-1", UserID: "user-1", Status: "active"},
		},
	}
	shipping := &fakeShippingForPlaceOrder{
		quotes: map[string]shippingclient.ShippingQuote{
			"quote-1": {
				ID: "quote-1", CustomerID: "cust-1", MethodID: "method-1",
				DestinationSnapshot: shippingclient.DestinationSnapshot{Label: "Home", FullName: "John", Street: "123 Main", City: "NYC", State: "NY", Zip: "10001", Country: "USA", Phone: "555-1234"},
				ShippingCostCents:   1000, EstimatedMinDays: 2, EstimatedMaxDays: 5, Status: "valid", ExpiresAt: "2099-01-01T00:00:00Z",
			},
		},
	}
	idem := &fakeIdempotencyForPlaceOrder{keys: make(map[string]string)}
	repo := memory.New()

	ctr := &contracts.Contracts{
		CatalogClient:  catalog,
		CustomerClient: customer,
		ShippingClient: shipping,
	}
	_ = ctr.Verify()

	svc := application.NewService(repo, ctr, idem, metrics.Noop{}, time.Hour)
	adapter := New(svc)

	t.Run("places an order successfully", func(t *testing.T) {
		got, err := adapter.PlaceOrder(t.Context(), orderclient.PlaceOrderRequest{
			UserID:              "user-1",
			IdempotencyKey:      "idem-1",
			Items:               []orderclient.LineItem{{ProductID: "prod-1", Quantity: 1}},
			ShippingQuoteID:     "quote-1",
			ShippingMethodID:    "method-1",
			DestinationSnapshot: orderclient.DestinationSnapshot{Label: "Home", FullName: "John", Street: "123 Main", City: "NYC", State: "NY", Zip: "10001", Country: "USA", Phone: "555-1234"},
			ShippingCostCents:   1000,
		})
		require.NoError(t, err)
		assert.Equal(t, "cust-1", got.CustomerID)
		assert.Equal(t, "user-1", got.UserID)
		assert.Equal(t, int64(1500), got.TotalCents) // 500 + 1000 shipping
	})

	t.Run("translates insufficient stock into client sentinel", func(t *testing.T) {
		catalog.err = catalogclient.ErrInsufficientStock
		_, err := adapter.PlaceOrder(t.Context(), orderclient.PlaceOrderRequest{
			UserID:              "user-1",
			IdempotencyKey:      "idem-2",
			Items:               []orderclient.LineItem{{ProductID: "prod-1", Quantity: 100}},
			ShippingQuoteID:     "quote-1",
			ShippingMethodID:    "method-1",
			DestinationSnapshot: orderclient.DestinationSnapshot{Label: "Home", FullName: "John", Street: "123 Main", City: "NYC", State: "NY", Zip: "10001", Country: "USA", Phone: "555-1234"},
			ShippingCostCents:   1000,
		})
		assert.ErrorIs(t, err, orderclient.ErrInsufficientStock)
	})

	t.Run("translates product not found into client sentinel", func(t *testing.T) {
		catalog.err = nil
		_, err := adapter.PlaceOrder(t.Context(), orderclient.PlaceOrderRequest{
			UserID:              "user-1",
			IdempotencyKey:      "idem-3",
			Items:               []orderclient.LineItem{{ProductID: "prod-missing", Quantity: 1}},
			ShippingQuoteID:     "quote-1",
			ShippingMethodID:    "method-1",
			DestinationSnapshot: orderclient.DestinationSnapshot{Label: "Home", FullName: "John", Street: "123 Main", City: "NYC", State: "NY", Zip: "10001", Country: "USA", Phone: "555-1234"},
			ShippingCostCents:   1000,
		})
		assert.ErrorIs(t, err, orderclient.ErrProductNotFound)
	})

	t.Run("translates idempotency conflict into client sentinel", func(t *testing.T) {
		catalog.err = nil
		idem.keys["idem-4"] = "fp"
		_, err := adapter.PlaceOrder(t.Context(), orderclient.PlaceOrderRequest{
			UserID:              "user-1",
			IdempotencyKey:      "idem-4",
			Items:               []orderclient.LineItem{{ProductID: "prod-1", Quantity: 1}},
			ShippingQuoteID:     "quote-1",
			ShippingMethodID:    "method-1",
			DestinationSnapshot: orderclient.DestinationSnapshot{Label: "Home", FullName: "John", Street: "123 Main", City: "NYC", State: "NY", Zip: "10001", Country: "USA", Phone: "555-1234"},
			ShippingCostCents:   1000,
		})
		assert.ErrorIs(t, err, orderclient.ErrIdempotencyConflict)
	})

	t.Run("passes through unexpected errors from catalog", func(t *testing.T) {
		boom := errors.New("boom")
		catalog.err = boom
		_, err := adapter.PlaceOrder(t.Context(), orderclient.PlaceOrderRequest{
			UserID:              "user-1",
			IdempotencyKey:      "idem-5",
			Items:               []orderclient.LineItem{{ProductID: "prod-1", Quantity: 1}},
			ShippingQuoteID:     "quote-1",
			ShippingMethodID:    "method-1",
			DestinationSnapshot: orderclient.DestinationSnapshot{Label: "Home", FullName: "John", Street: "123 Main", City: "NYC", State: "NY", Zip: "10001", Country: "USA", Phone: "555-1234"},
			ShippingCostCents:   1000,
		})
		require.ErrorIs(t, err, boom)
	})
}
