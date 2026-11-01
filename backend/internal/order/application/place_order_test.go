package application

import (
	"context"
	"errors"
	customerclient "minimart/internal/customer/presentation/module/client"
	"minimart/internal/order/domain"
	orderclient "minimart/internal/order/presentation/module/client"
	"minimart/pkg/idempotency"
	"minimart/pkg/metrics"
	"minimart/pkg/module/contracts"
	"testing"
	"time"

	addressclient "minimart/internal/address/presentation/module/client"
	catalogclient "minimart/internal/catalog/presentation/module/client"
	categoryclient "minimart/internal/category/presentation/module/client"
	shippingclient "minimart/internal/shipping/presentation/module/client"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fakeRepo is a hand-written test double for the order Repository port. The order
// it stores carries the domain events produced by the aggregate, which the real
// repository would persist to the outbox.
type fakeRepo struct {
	saved          domain.Order
	byKey          map[string]domain.Order
	saveErr        error
	getErr         error
	updateErr      error
	updateExpected domain.Status
	updateCalls    int
}

func (f *fakeRepo) Save(_ context.Context, o domain.Order) error {
	f.saved = o
	if o.IdempotencyKey != "" {
		if f.byKey == nil {
			f.byKey = map[string]domain.Order{}
		}
		f.byKey[o.IdempotencyKey] = o
	}
	return f.saveErr
}

func (f *fakeRepo) GetByID(context.Context, string) (domain.Order, error) {
	return f.saved, f.getErr
}

func (f *fakeRepo) GetByIdempotencyKey(_ context.Context, key string) (domain.Order, error) {
	if o, ok := f.byKey[key]; ok {
		return o, nil
	}
	return domain.Order{}, domain.ErrOrderNotFound
}

func (f *fakeRepo) Update(_ context.Context, o domain.Order, expected domain.Status) error {
	f.saved = o
	f.updateExpected = expected
	f.updateCalls++
	return f.updateErr
}

func (f *fakeRepo) ListByUserID(context.Context, string, int, int) ([]domain.Order, error) {
	return nil, nil
}

func (f *fakeRepo) CountByUserID(context.Context, string) (int, error) {
	return 0, nil
}

func (f *fakeRepo) ListPlacedBefore(context.Context, time.Time, int) ([]domain.Order, error) {
	if f.saved.ID != "" && f.saved.Status == domain.StatusPlaced {
		return []domain.Order{f.saved}, nil
	}
	return nil, nil
}

// fakeModules impersonates the entire catalog domain via the published contract
// and the customer domain via the customer contract.
type fakeModules struct {
	shippingclient.ShippingClient
	addressclient.AddressClient
	products   map[string]catalogclient.Product
	stock      map[string]int
	reserveErr error
	releaseErr error
	reserved   []string
	released   []string
}

func (f *fakeModules) GetProduct(_ context.Context, id string) (catalogclient.Product, error) {
	p, ok := f.products[id]
	if !ok {
		return catalogclient.Product{}, catalogclient.ErrProductNotFound
	}
	return p, nil
}

func (f *fakeModules) GetCategory(_ context.Context, id string) (categoryclient.CategoryDTO, error) {
	return categoryclient.CategoryDTO{ID: id, Name: "Test Category"}, nil
}

func (f *fakeModules) ReserveStock(_ context.Context, orderID string, items []catalogclient.LineItem) error {
	f.reserved = append(f.reserved, orderID)
	if f.reserveErr != nil {
		return f.reserveErr
	}
	for _, it := range items {
		if avail, ok := f.stock[it.ProductID]; ok && avail < it.Quantity {
			return catalogclient.ErrInsufficientStock
		}
	}
	for _, it := range items {
		if _, ok := f.stock[it.ProductID]; ok {
			f.stock[it.ProductID] -= it.Quantity
		}
	}
	return nil
}

func (f *fakeModules) ReleaseStock(_ context.Context, orderID string) error {
	f.released = append(f.released, orderID)
	return f.releaseErr
}

func (f *fakeModules) EnsureCustomer(_ context.Context, userID string) (customerclient.Customer, error) {
	if userID == "" {
		return customerclient.Customer{}, errors.New("customer: user_id é obrigatório")
	}
	return customerclient.Customer{ID: "cust-" + userID, UserID: userID}, nil
}

func (f *fakeModules) GetCustomerByID(_ context.Context, id string) (customerclient.Customer, error) {
	return customerclient.Customer{ID: id, UserID: "user-" + id}, nil
}

func (f *fakeModules) GetCustomerByUserID(_ context.Context, userID string) (customerclient.Customer, error) {
	return customerclient.Customer{ID: "cust-" + userID, UserID: userID}, nil
}

func (f *fakeModules) PlaceOrder(_ context.Context, req orderclient.PlaceOrderRequest) (orderclient.Order, error) {
	return orderclient.Order{}, errors.New("not implemented in test fake")
}

func (f *fakeModules) GetOrder(_ context.Context, id string) (orderclient.Order, error) {
	return orderclient.Order{}, nil
}

// memIdempotency is a minimal in-memory idempotency.Store for tests.
type memIdempotency struct {
	records map[string]idempotency.Record
}

func newMemIdempotency() *memIdempotency {
	return &memIdempotency{records: map[string]idempotency.Record{}}
}

func (s *memIdempotency) Get(_ context.Context, key string) (idempotency.Record, bool, error) {
	r, ok := s.records[key]
	return r, ok, nil
}

func (s *memIdempotency) Reserve(_ context.Context, key, fp string, _ time.Duration) (bool, error) {
	if _, ok := s.records[key]; ok {
		return false, nil
	}
	s.records[key] = idempotency.Record{Fingerprint: fp}
	return true, nil
}

func (s *memIdempotency) Complete(_ context.Context, key, fp, resourceID string, _ time.Duration) error {
	s.records[key] = idempotency.Record{Fingerprint: fp, ResourceID: resourceID}
	return nil
}

func (s *memIdempotency) Release(_ context.Context, key, _ string) error {
	delete(s.records, key)
	return nil
}

// newSvc builds a service with a fresh in-memory idempotency store, for tests that
// do not exercise idempotency directly.
func newSvc(repo domain.Repository, modules contracts.Modules, m metrics.Metrics) *Service {
	return NewService(repo, modules, newMemIdempotency(), m, time.Hour)
}

// countingMetrics records order results while no-op'ing every other signal.
type countingMetrics struct {
	metrics.Noop
	placed         []string
	cancelled      []string
	amount         []float64
	paymentApplied []string
}

func (c *countingMetrics) IncOrderPlaced(status string)     { c.placed = append(c.placed, status) }
func (c *countingMetrics) IncOrderCancelled(status string)  { c.cancelled = append(c.cancelled, status) }
func (c *countingMetrics) ObserveOrderAmount(cents float64) { c.amount = append(c.amount, cents) }
func (c *countingMetrics) IncOrderPaymentApplied(status string) {
	c.paymentApplied = append(c.paymentApplied, status)
}

func testCatalog() *fakeModules {
	return &fakeModules{products: map[string]catalogclient.Product{
		"p1": {ID: "p1", Name: "Widget", PriceCents: 500},
		"p2": {ID: "p2", Name: "Gadget", PriceCents: 250},
	}}
}

// findPlaced returns the OrderPlacedEvent recorded on the aggregate, which the
// repository would write to the outbox.
func findPlaced(t *testing.T, o domain.Order) domain.OrderPlacedEvent {
	t.Helper()
	for _, ev := range o.Events() {
		if e, ok := ev.(domain.OrderPlacedEvent); ok {
			return e
		}
	}
	t.Fatal("no OrderPlacedEvent recorded on the order")
	return domain.OrderPlacedEvent{}
}

func TestPlaceOrder(t *testing.T) {
	tests := []struct {
		name      string
		req       PlaceOrderRequest
		wantTotal int64
		wantErr   error
	}{
		{
			name:      "two items sums line totals and records one event",
			req:       PlaceOrderRequest{UserID: "u1", IdempotencyKey: "k1", Items: []PlaceOrderItem{{"p1", 2}, {"p2", 1}}},
			wantTotal: 1250,
		},
		{
			name:    "no items",
			req:     PlaceOrderRequest{UserID: "u1", IdempotencyKey: "k1"},
			wantErr: domain.ErrNoItems,
		},
		{
			name:    "unknown product",
			req:     PlaceOrderRequest{UserID: "u1", IdempotencyKey: "k1", Items: []PlaceOrderItem{{"nope", 1}}},
			wantErr: catalogclient.ErrProductNotFound,
		},
		{
			name:    "non-positive quantity",
			req:     PlaceOrderRequest{UserID: "u1", IdempotencyKey: "k1", Items: []PlaceOrderItem{{"p1", 0}}},
			wantErr: domain.ErrInvalidQuantity,
		},
		{
			name:    "missing idempotency key",
			req:     PlaceOrderRequest{UserID: "u1", Items: []PlaceOrderItem{{"p1", 1}}},
			wantErr: ErrIdempotencyKeyRequired,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			repo := &fakeRepo{}
			svc := newSvc(repo, testCatalog(), metrics.Noop{})
			got, err := svc.PlaceOrder(t.Context(), tt.req)
			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
				assert.Empty(t, repo.saved.ID, "nothing is persisted on failure")
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.wantTotal, got.TotalCents)
			assert.Equal(t, got.ID, repo.saved.ID)

			event := findPlaced(t, repo.saved)
			assert.Equal(t, got.ID, event.OrderID)
			assert.Equal(t, "u1", event.UserID)
			assert.Equal(t, tt.wantTotal, event.TotalCents)
			assert.Equal(t, len(tt.req.Items), event.ItemCount)
		})
	}
}

func TestPlaceOrderReservesStock(t *testing.T) {
	repo := &fakeRepo{}
	modules := testCatalog()
	modules.stock = map[string]int{"p1": 5}
	svc := newSvc(repo, modules, metrics.Noop{})

	got, err := svc.PlaceOrder(t.Context(), PlaceOrderRequest{
		UserID: "u1", IdempotencyKey: "k1", Items: []PlaceOrderItem{{"p1", 2}},
	})
	require.NoError(t, err)
	assert.Equal(t, []string{got.ID}, modules.reserved, "stock is reserved for the new order")
	assert.Equal(t, 3, modules.stock["p1"], "reserved units are removed from stock")

	t.Run("insufficient stock rejects placement", func(t *testing.T) {
		repo := &fakeRepo{}
		modules := testCatalog()
		modules.stock = map[string]int{"p1": 1}
		svc := newSvc(repo, modules, metrics.Noop{})

		_, err := svc.PlaceOrder(t.Context(), PlaceOrderRequest{
			UserID: "u1", IdempotencyKey: "k1", Items: []PlaceOrderItem{{"p1", 5}},
		})
		require.ErrorIs(t, err, catalogclient.ErrInsufficientStock)
		assert.Empty(t, repo.saved.ID, "no order is saved when stock is short")
	})
}

func TestPlaceOrderPersistenceFailureReleasesStock(t *testing.T) {
	boom := errors.New("disk on fire")
	repo := &fakeRepo{saveErr: boom}
	modules := testCatalog()
	modules.stock = map[string]int{"p1": 5}
	m := &countingMetrics{}
	svc := newSvc(repo, modules, m)

	_, err := svc.PlaceOrder(t.Context(), PlaceOrderRequest{
		UserID: "u1", IdempotencyKey: "k1", Items: []PlaceOrderItem{{"p1", 1}},
	})
	require.ErrorIs(t, err, boom)
	assert.Equal(t, []string{metrics.StatusError}, m.placed)
	require.Len(t, modules.reserved, 1)
	assert.Equal(t, modules.reserved, modules.released, "the reservation is released when saving fails")
}

func TestPlaceOrderRecordsMetrics(t *testing.T) {
	m := &countingMetrics{}
	svc := newSvc(&fakeRepo{}, testCatalog(), m)

	_, err := svc.PlaceOrder(t.Context(), PlaceOrderRequest{
		UserID: "u1", IdempotencyKey: "k1", Items: []PlaceOrderItem{{"p1", 2}},
	})
	require.NoError(t, err)
	assert.Equal(t, []string{metrics.StatusSuccess}, m.placed)
	assert.Equal(t, []float64{1000}, m.amount)
}

func TestPlaceOrderIsIdempotent(t *testing.T) {
	t.Run("a repeated key replays the stored order without re-reserving", func(t *testing.T) {
		t.Parallel()
		repo := &fakeRepo{}
		modules := testCatalog()
		modules.stock = map[string]int{"p1": 5}
		svc := newSvc(repo, modules, metrics.Noop{})
		req := PlaceOrderRequest{UserID: "u1", IdempotencyKey: "k1", Items: []PlaceOrderItem{{"p1", 2}}}

		first, err := svc.PlaceOrder(t.Context(), req)
		require.NoError(t, err)
		second, err := svc.PlaceOrder(t.Context(), req)
		require.NoError(t, err)

		assert.Equal(t, first.ID, second.ID, "the same order is returned")
		assert.Len(t, modules.reserved, 1, "stock is reserved only once")
		assert.Equal(t, 3, modules.stock["p1"], "stock is decremented only once")
	})

	t.Run("the same key with different parameters is a conflict", func(t *testing.T) {
		t.Parallel()
		repo := &fakeRepo{}
		svc := newSvc(repo, testCatalog(), metrics.Noop{})

		_, err := svc.PlaceOrder(t.Context(), PlaceOrderRequest{
			UserID: "u1", IdempotencyKey: "k1", Items: []PlaceOrderItem{{"p1", 2}},
		})
		require.NoError(t, err)

		_, err = svc.PlaceOrder(t.Context(), PlaceOrderRequest{
			UserID: "u1", IdempotencyKey: "k1", Items: []PlaceOrderItem{{"p1", 3}},
		})
		require.ErrorIs(t, err, domain.ErrIdempotencyConflict)
	})
}

func TestPlaceOrderPersistsShippingSnapshot(t *testing.T) {
	repo := &fakeRepo{}
	svc := newSvc(repo, testCatalog(), metrics.Noop{})

	dest := domain.DestinationSnapshot{
		FullName: "Ada Lovelace",
		Street:   "1 Main St",
		City:     "Testville",
		Zip:      "12345",
		Country:  "BR",
	}
	got, err := svc.PlaceOrder(t.Context(), PlaceOrderRequest{
		UserID:              "u1",
		IdempotencyKey:      "k1",
		Items:               []PlaceOrderItem{{"p1", 1}},
		ShippingQuoteID:     "quote-1",
		ShippingMethodID:    "method-1",
		ShippingMethodName:  "Standard",
		DestinationSnapshot: dest,
		ShippingCostCents:   250,
	})
	require.NoError(t, err)

	assert.Equal(t, int64(750), got.TotalCents, "the shipping cost is folded into the total")
	assert.Equal(t, int64(250), got.ShippingCostCents)
	assert.Equal(t, "quote-1", got.ShippingQuoteID)
	assert.Equal(t, "method-1", got.ShippingMethodID)
	assert.Equal(t, "Standard", got.ShippingMethodName)
	assert.Equal(t, dest, got.DestinationSnapshot)

	// The stored order carries the same snapshot: it is what the payment
	// consumer reloads to hand the shipping data over to the shipping domain.
	assert.Equal(t, got.ShippingQuoteID, repo.saved.ShippingQuoteID)
	assert.Equal(t, got.ShippingMethodID, repo.saved.ShippingMethodID)
	assert.Equal(t, got.ShippingCostCents, repo.saved.ShippingCostCents)
	assert.Equal(t, dest, repo.saved.DestinationSnapshot)
}

func TestPlaceOrderWithoutShippingHasNoSnapshot(t *testing.T) {
	repo := &fakeRepo{}
	svc := newSvc(repo, testCatalog(), metrics.Noop{})

	got, err := svc.PlaceOrder(t.Context(), PlaceOrderRequest{
		UserID: "u1", IdempotencyKey: "k1", Items: []PlaceOrderItem{{"p1", 1}},
	})
	require.NoError(t, err)

	assert.Equal(t, int64(500), got.TotalCents)
	assert.Empty(t, got.ShippingQuoteID)
	assert.Empty(t, got.ShippingMethodID)
	assert.Empty(t, got.ShippingMethodName)
	assert.Zero(t, got.ShippingCostCents)
	assert.Equal(t, domain.DestinationSnapshot{}, got.DestinationSnapshot)
}

// failingIdempotency is an idempotency.Store that is unavailable: every call
// fails, so the placement falls back to the orders table.
type failingIdempotency struct{}

func (failingIdempotency) Get(context.Context, string) (idempotency.Record, bool, error) {
	return idempotency.Record{}, false, errors.New("idempotency store unavailable")
}

func (failingIdempotency) Reserve(context.Context, string, string, time.Duration) (bool, error) {
	return false, errors.New("idempotency store unavailable")
}

func (failingIdempotency) Complete(context.Context, string, string, string, time.Duration) error {
	return errors.New("idempotency store unavailable")
}

func (failingIdempotency) Release(context.Context, string, string) error {
	return errors.New("idempotency store unavailable")
}

func TestPlaceOrderFallsBackToTheDatabaseWithoutTheCache(t *testing.T) {
	repo := &fakeRepo{}
	modules := testCatalog()
	modules.stock = map[string]int{"p1": 5}
	svc := NewService(repo, modules, failingIdempotency{}, metrics.Noop{}, time.Hour)

	req := PlaceOrderRequest{
		UserID:             "u1",
		IdempotencyKey:     "k1",
		Items:              []PlaceOrderItem{{"p1", 1}},
		ShippingQuoteID:    "quote-1",
		ShippingMethodID:   "method-1",
		ShippingMethodName: "Standard",
		ShippingCostCents:  250,
	}
	first, err := svc.PlaceOrder(t.Context(), req)
	require.NoError(t, err)

	second, err := svc.PlaceOrder(t.Context(), req)
	require.NoError(t, err, "the stored order is replayed when the cache is unavailable")
	assert.Equal(t, first.ID, second.ID)
}
