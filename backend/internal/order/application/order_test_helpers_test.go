package application

import (
	"context"
	"errors"
	"minimart/internal/order/domain"
	"minimart/internal/order/infrastructure/memory"
	"minimart/pkg/idempotency"
	"minimart/pkg/metrics"
	"minimart/pkg/module/contracts"
	"testing"
	"time"

	addressclient "minimart/internal/address/presentation/module/client"
	catalogclient "minimart/internal/catalog/presentation/module/client"
	categoryclient "minimart/internal/category/presentation/module/client"
	customerclient "minimart/internal/customer/presentation/module/client"
	orderclient "minimart/internal/order/presentation/module/client"
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
	if f.saveErr != nil {
		return f.saveErr
	}
	f.saved = o
	if o.IdempotencyKey != "" {
		if f.byKey == nil {
			f.byKey = map[string]domain.Order{}
		}
		f.byKey[o.IdempotencyKey] = o
	}
	return nil
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
	f.updateExpected = expected
	f.updateCalls++
	if f.updateErr != nil {
		return f.updateErr
	}
	f.saved = o
	return nil
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

// fakeModules impersonates the catalog and customer contracts so unit tests never
// reach another domain's implementation. Stock is tracked in memory.
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

// testCatalog returns a fake catalog with the products used by the features.
func testCatalog() *fakeModules {
	return &fakeModules{products: map[string]catalogclient.Product{
		"p1": {ID: "p1", Name: "Widget", PriceCents: 500},
		"p2": {ID: "p2", Name: "Gadget", PriceCents: 250},
	}}
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

// failingIdempotency is an idempotency.Store that is unavailable: every call
// fails, so placements fall back to the orders table.
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

// newSvc builds a service with a fresh in-memory idempotency store.
func newSvc(repo domain.Repository, modules contracts.Modules, m metrics.Metrics) *Service {
	return NewService(repo, modules, newMemIdempotency(), m, time.Hour)
}

// newTestService builds a service over an empty in-memory order repository.
func newTestService(t *testing.T) (*Service, *memory.Repository) {
	t.Helper()
	repo := memory.New()
	return NewService(repo, testCatalog(), newMemIdempotency(), metrics.Noop{}, time.Hour), repo
}

// placedOrder returns the canonical placed order used by feature fixtures.
func placedOrder(t *testing.T) domain.Order {
	t.Helper()
	o, err := domain.NewOrder("o1", "u1", "c1", "", []domain.Item{
		{ProductID: "p1", Name: "Widget", UnitPriceCents: 500, Quantity: 1},
	})
	require.NoError(t, err)
	return o
}

// featureDestination is the delivery snapshot named by the features.
func featureDestination() domain.DestinationSnapshot {
	return domain.DestinationSnapshot{
		FullName: "Ada Lovelace",
		Street:   "1 Main St",
		City:     "Testville",
		Country:  "BR",
	}
}

// placedOrderWithShipping returns a placed order carrying the shipping snapshot.
func placedOrderWithShipping(t *testing.T) domain.Order {
	t.Helper()
	o := placedOrder(t)
	o.WithShipping(domain.ShippingSnapshot{
		QuoteID:     "q1",
		MethodID:    "m1",
		MethodName:  "Standard",
		CostCents:   250,
		Destination: featureDestination(),
	})
	return o
}

// newStoredOrder builds an order for fixtures at the given status and timestamp.
func newStoredOrder(t *testing.T, id, userID string, status domain.Status, createdAt time.Time) domain.Order {
	t.Helper()
	o, err := domain.NewOrder(id, userID, "", "", []domain.Item{
		{ProductID: "p1", Name: "Widget", UnitPriceCents: 500, Quantity: 1},
	})
	require.NoError(t, err)
	o.Status = status
	o.CreatedAt = createdAt
	return o
}

// saveOrder persists a fixture in the in-memory repository.
func saveOrder(t *testing.T, repo *memory.Repository, o domain.Order) domain.Order {
	t.Helper()
	require.NoError(t, repo.Save(t.Context(), o))
	return o
}

// stalePlacedRepo returns a stale placed snapshot regardless of the stored state,
// simulating a status change that lands between the scan's read and its write.
type stalePlacedRepo struct {
	*memory.Repository
	stale []domain.Order
}

func (r *stalePlacedRepo) ListPlacedBefore(context.Context, time.Time, int) ([]domain.Order, error) {
	return r.stale, nil
}

// validPlaceOrderRequest is the canonical placement payload: one Widget.
func validPlaceOrderRequest(userID, idempotencyKey string) PlaceOrderRequest {
	return PlaceOrderRequest{
		UserID:         userID,
		IdempotencyKey: idempotencyKey,
		Items:          []PlaceOrderItem{{ProductID: "p1", Quantity: 1}},
	}
}

// findPlaced returns the single OrderPlacedEvent recorded on the aggregate.
func findPlaced(t *testing.T, o domain.Order) domain.OrderPlacedEvent {
	t.Helper()
	var found []domain.OrderPlacedEvent
	for _, ev := range o.Events() {
		if e, ok := ev.(domain.OrderPlacedEvent); ok {
			found = append(found, e)
		}
	}
	require.Len(t, found, 1, "exactly one OrderPlacedEvent is recorded")
	return found[0]
}

// findCancelled returns the single OrderCancelledEvent recorded on the aggregate.
func findCancelled(t *testing.T, o domain.Order) domain.OrderCancelledEvent {
	t.Helper()
	var found []domain.OrderCancelledEvent
	for _, ev := range o.Events() {
		if e, ok := ev.(domain.OrderCancelledEvent); ok {
			found = append(found, e)
		}
	}
	require.Len(t, found, 1, "exactly one OrderCancelledEvent is recorded")
	return found[0]
}

// findReleased returns the single OrderStockReleaseEvent recorded on the aggregate.
func findReleased(t *testing.T, o domain.Order) domain.OrderStockReleaseEvent {
	t.Helper()
	var found []domain.OrderStockReleaseEvent
	for _, ev := range o.Events() {
		if e, ok := ev.(domain.OrderStockReleaseEvent); ok {
			found = append(found, e)
		}
	}
	require.Len(t, found, 1, "exactly one OrderStockReleaseEvent is recorded")
	return found[0]
}

// findPaid returns the single OrderPaidEvent recorded on the aggregate.
func findPaid(t *testing.T, o domain.Order) domain.OrderPaidEvent {
	t.Helper()
	var found []domain.OrderPaidEvent
	for _, ev := range o.Events() {
		if e, ok := ev.(domain.OrderPaidEvent); ok {
			found = append(found, e)
		}
	}
	require.Len(t, found, 1, "exactly one OrderPaidEvent is recorded")
	return found[0]
}

// countCancelled returns how many OrderCancelledEvents the aggregate records.
func countCancelled(o domain.Order) int {
	n := 0
	for _, ev := range o.Events() {
		if _, ok := ev.(domain.OrderCancelledEvent); ok {
			n++
		}
	}
	return n
}

// countReleased returns how many OrderStockReleaseEvents the aggregate records.
func countReleased(o domain.Order) int {
	n := 0
	for _, ev := range o.Events() {
		if _, ok := ev.(domain.OrderStockReleaseEvent); ok {
			n++
		}
	}
	return n
}

// countPaid returns how many OrderPaidEvents the aggregate records.
func countPaid(o domain.Order) int {
	n := 0
	for _, ev := range o.Events() {
		if _, ok := ev.(domain.OrderPaidEvent); ok {
			n++
		}
	}
	return n
}

func assertShippingSnapshot(t *testing.T, got domain.Order) {
	t.Helper()
	assert.Equal(t, "q1", got.ShippingQuoteID)
	assert.Equal(t, "m1", got.ShippingMethodID)
	assert.Equal(t, int64(250), got.ShippingCostCents)
	assert.Equal(t, featureDestination(), got.DestinationSnapshot)
}
