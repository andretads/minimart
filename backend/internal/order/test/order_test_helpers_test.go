//go:build integration

// Package test holds the order domain's integration tests.
package test

import (
	"context"
	"crypto/rand"
	"errors"
	"minimart/internal/order/application"
	"minimart/internal/order/domain"
	"minimart/internal/order/infrastructure/postgres"
	"minimart/pkg/idempotency"
	"minimart/pkg/metrics"
	"strconv"
	"sync"
	"testing"
	"time"

	addressclient "minimart/internal/address/presentation/module/client"
	catalogclient "minimart/internal/catalog/presentation/module/client"
	categoryclient "minimart/internal/category/presentation/module/client"
	customerclient "minimart/internal/customer/presentation/module/client"
	orderclient "minimart/internal/order/presentation/module/client"
	shippingclient "minimart/internal/shipping/presentation/module/client"

	outboxpkg "minimart/pkg/outbox"
	outboxpg "minimart/pkg/outbox/postgres"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"
)

// uniqueOrderID returns an id unlikely to collide with a previous run's rows.
// orders.id is a plain primary key (no upsert), so a fixed id would fail on the
// second run.
func uniqueOrderID() string {
	return "it-order-" + rand.Text()
}

// newOrderRepo connects to the real database and returns the adapter. The read
// and write pools point at the same database, and the repository writes domain
// events to the shared outbox via the real Publisher.
func newOrderRepo(t *testing.T) (*postgres.OrderRepository, *pgxpool.Pool) {
	t.Helper()

	pool, err := pgxpool.New(t.Context(), testDSN)
	require.NoError(t, err)
	t.Cleanup(pool.Close)

	store := outboxpg.NewStore(pool)
	return postgres.NewOrderRepository(pool, pool, outboxpkg.NewPublisher(store)), pool
}

// cleanup deletes the order, its items, and any outbox events keyed to it after
// the test. t.Context() is cancelled before cleanup runs, so use a fresh context.
func cleanup(t *testing.T, pool *pgxpool.Pool, id string) {
	t.Helper()
	t.Cleanup(func() {
		//nolint:usetesting // t.Context() is cancelled before cleanup runs
		ctx := context.Background()
		_, _ = pool.Exec(ctx, "DELETE FROM order_items WHERE order_id = $1", id)
		_, _ = pool.Exec(ctx, "DELETE FROM orders WHERE id = $1", id)
		_, _ = pool.Exec(ctx, "DELETE FROM outbox WHERE message_key = $1", id)
	})
}

// outboxTopics returns the topics of the outbox rows keyed to the given order id.
func outboxTopics(ctx context.Context, t *testing.T, pool *pgxpool.Pool, key string) []string {
	t.Helper()
	rows, err := pool.Query(ctx, "SELECT topic FROM outbox WHERE message_key = $1", key)
	require.NoError(t, err)
	defer rows.Close()
	var topics []string
	for rows.Next() {
		var topic string
		require.NoError(t, rows.Scan(&topic))
		topics = append(topics, topic)
	}
	require.NoError(t, rows.Err())
	return topics
}

// countOutbox returns how many outbox rows are keyed to the given order id.
func countOutbox(ctx context.Context, t *testing.T, pool *pgxpool.Pool, key string) int {
	t.Helper()
	var n int
	require.NoError(t, pool.QueryRow(ctx, "SELECT count(*) FROM outbox WHERE message_key = $1", key).Scan(&n))
	return n
}

// outboxBody returns the payload of the outbox row written for the given order
// id and topic.
func outboxBody(ctx context.Context, t *testing.T, pool *pgxpool.Pool, key string, topic string) []byte {
	t.Helper()
	var body []byte
	err := pool.QueryRow(ctx, "SELECT body FROM outbox WHERE message_key = $1 AND topic = $2", key, topic).Scan(&body)
	require.NoError(t, err)
	return body
}

// countOutboxTopic returns how many outbox rows for key carry the given topic.
func countOutboxTopic(ctx context.Context, t *testing.T, pool *pgxpool.Pool, key, topic string) int {
	t.Helper()
	var n int
	require.NoError(t, pool.QueryRow(ctx,
		"SELECT count(*) FROM outbox WHERE message_key = $1 AND topic = $2", key, topic).Scan(&n))
	return n
}

// clearOutbox removes the outbox rows keyed to an order, so a later assertion
// only sees the events written by the operation under test.
func clearOutbox(t *testing.T, pool *pgxpool.Pool, id string) {
	t.Helper()
	_, err := pool.Exec(t.Context(), "DELETE FROM outbox WHERE message_key = $1", id)
	require.NoError(t, err)
}

// setCreatedAt rewrites the stored creation time so ordering and stale-order
// scenarios are deterministic across the shared database.
func setCreatedAt(t *testing.T, pool *pgxpool.Pool, id string, createdAt time.Time) {
	t.Helper()
	_, err := pool.Exec(t.Context(), "UPDATE orders SET created_at = $1 WHERE id = $2", createdAt, id)
	require.NoError(t, err)
}

// customerIDFor derives the customer id the fake customer contract returns for a
// user; uniqueUser inserts the matching row so the orders FK holds.
func customerIDFor(userID string) string { return "it-cust-" + userID }

// uniqueUser creates a fresh customer row and returns its user id, so scenarios
// never share orders and cleanup never truncates a shared table.
func uniqueUser(t *testing.T, pool *pgxpool.Pool) string {
	t.Helper()
	userID := "it-user-" + rand.Text()
	_, err := pool.Exec(t.Context(),
		"INSERT INTO customers (id, user_id) VALUES ($1, $2)", customerIDFor(userID), userID)
	require.NoError(t, err)
	t.Cleanup(func() {
		//nolint:usetesting // t.Context() is cancelled before cleanup runs
		ctx := context.Background()
		_, _ = pool.Exec(ctx, "DELETE FROM customers WHERE user_id = $1", userID)
	})
	return userID
}

// fakeModules impersonates the catalog and customer contracts. The order domain
// under test still uses its real Postgres repository and constraints.
type fakeModules struct {
	shippingclient.ShippingClient
	addressclient.AddressClient

	products   map[string]catalogclient.Product
	stock      map[string]int
	customers  map[string]customerclient.Customer
	reserveErr error
	releaseErr error
	reserved   []string
	released   []string
}

func newFakeModules() *fakeModules {
	return &fakeModules{
		products: map[string]catalogclient.Product{
			"p1": {ID: "p1", Name: "Widget", PriceCents: 500},
			"p2": {ID: "p2", Name: "Gadget", PriceCents: 250},
		},
		stock:     map[string]int{"p1": 1000, "p2": 1000},
		customers: map[string]customerclient.Customer{},
	}
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
	if c, ok := f.customers[userID]; ok {
		return c, nil
	}
	c := customerclient.Customer{ID: customerIDFor(userID), UserID: userID, DisplayName: "Test", Status: "active"}
	f.customers[userID] = c
	return c, nil
}

func (f *fakeModules) GetCustomerByID(_ context.Context, id string) (customerclient.Customer, error) {
	return customerclient.Customer{ID: id, UserID: "user-" + id}, nil
}

func (f *fakeModules) GetCustomerByUserID(_ context.Context, userID string) (customerclient.Customer, error) {
	return customerclient.Customer{ID: customerIDFor(userID), UserID: userID}, nil
}

func (f *fakeModules) PlaceOrder(_ context.Context, req orderclient.PlaceOrderRequest) (orderclient.Order, error) {
	return orderclient.Order{}, nil
}

func (f *fakeModules) GetOrder(_ context.Context, id string) (orderclient.Order, error) {
	return orderclient.Order{}, nil
}

// unavailableIdem is an idempotency.Store whose backend is down, so placements
// fall back to the orders table's unique index.
type unavailableIdem struct{}

var errIdempotencyStoreUnavailable = errors.New("idempotency store unavailable")

func (unavailableIdem) Get(context.Context, string) (idempotency.Record, bool, error) {
	return idempotency.Record{}, false, errIdempotencyStoreUnavailable
}

func (unavailableIdem) Reserve(context.Context, string, string, time.Duration) (bool, error) {
	return false, errIdempotencyStoreUnavailable
}

func (unavailableIdem) Complete(context.Context, string, string, string, time.Duration) error {
	return errIdempotencyStoreUnavailable
}

func (unavailableIdem) Release(context.Context, string, string) error {
	return errIdempotencyStoreUnavailable
}

// failingSaveRepo delegates to the real repository but fails every save, which
// lets a test prove the use case releases the stock reservation.
type failingSaveRepo struct {
	*postgres.OrderRepository
	err error
}

func (r failingSaveRepo) Save(context.Context, domain.Order) error { return r.err }

// failingUpdateRepo delegates to the real repository but fails every update.
type failingUpdateRepo struct {
	*postgres.OrderRepository
	err error
}

func (r failingUpdateRepo) Update(context.Context, domain.Order, domain.Status) error { return r.err }

// staleListRepo returns a captured placed snapshot regardless of the stored
// state, simulating a concurrent status change before the auto-cancel write.
type staleListRepo struct {
	*postgres.OrderRepository
	stale []domain.Order
}

func (r *staleListRepo) ListPlacedBefore(context.Context, time.Time, int) ([]domain.Order, error) {
	return r.stale, nil
}

// placeOrderFingerprint mirrors the use case's idempotency fingerprint for the
// fields that define a placement request.
func placeOrderFingerprint(req application.PlaceOrderRequest) string {
	parts := make([]string, 0, 1+2*len(req.Items)+3)
	parts = append(parts, req.UserID)
	for _, it := range req.Items {
		parts = append(parts, it.ProductID, strconv.Itoa(it.Quantity))
	}
	parts = append(parts, req.ShippingQuoteID, req.ShippingMethodID, strconv.FormatInt(req.ShippingCostCents, 10))
	return idempotency.Fingerprint(parts...)
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

// newOrderService wires the order service over the real repository and the fake
// cross-domain contracts.
func newOrderService(t *testing.T) (*application.Service, *postgres.OrderRepository, *pgxpool.Pool, *fakeModules) {
	t.Helper()
	return newOrderServiceWithIdem(t, newMemIdem())
}

func newOrderServiceWithIdem(t *testing.T, idem idempotency.Store) (*application.Service, *postgres.OrderRepository, *pgxpool.Pool, *fakeModules) {
	t.Helper()
	repo, pool := newOrderRepo(t)
	modules := newFakeModules()
	svc := application.NewService(repo, modules, idem, metrics.Noop{}, time.Hour)
	return svc, repo, pool, modules
}

// seedOrder stores a placed order with the feature's canonical one-Widget line.
func seedOrder(t *testing.T, repo *postgres.OrderRepository, pool *pgxpool.Pool, id, userID string) domain.Order {
	t.Helper()
	cleanup(t, pool, id)
	o, err := domain.NewOrder(id, userID, "", "", []domain.Item{
		{ProductID: "p1", Name: "Widget", UnitPriceCents: 500, Quantity: 1},
	})
	require.NoError(t, err)
	require.NoError(t, repo.Save(t.Context(), o))
	stored, err := repo.GetByID(t.Context(), id)
	require.NoError(t, err)
	return stored
}

// seedOrderWithStatus stores a placed order and moves it to status through the
// real repository, re-reading first so only the transition's events are written.
func seedOrderWithStatus(t *testing.T, repo *postgres.OrderRepository, pool *pgxpool.Pool, id, userID string, status domain.Status) domain.Order {
	t.Helper()
	stored := seedOrder(t, repo, pool, id, userID)
	if status == domain.StatusPlaced {
		return stored
	}

	var next domain.Order
	switch status {
	case domain.StatusPaid:
		next = stored.MarkPaid()
	case domain.StatusPaymentFailed:
		next = stored.MarkPaymentFailed()
	case domain.StatusPaymentUnknown:
		next = stored.MarkPaymentUnknown()
	case domain.StatusCancelled:
		var err error
		next, err = stored.Cancel()
		require.NoError(t, err)
	default:
		t.Fatalf("unsupported seeded status %q", status)
	}
	require.NoError(t, repo.Update(t.Context(), next, domain.StatusPlaced))
	return next
}

// seedShippingOrder stores an order carrying the feature's shipping snapshot.
func seedShippingOrder(t *testing.T, repo *postgres.OrderRepository, pool *pgxpool.Pool, id, userID string) domain.Order {
	t.Helper()
	cleanup(t, pool, id)
	o, err := domain.NewOrder(id, userID, "", "", []domain.Item{
		{ProductID: "p1", Name: "Widget", UnitPriceCents: 500, Quantity: 1},
	})
	require.NoError(t, err)
	o.WithShipping(domain.ShippingSnapshot{
		QuoteID:     "q1",
		MethodID:    "m1",
		MethodName:  "Standard",
		CostCents:   250,
		Destination: featureDestination(),
	})
	require.NoError(t, repo.Save(t.Context(), o))
	stored, err := repo.GetByID(t.Context(), id)
	require.NoError(t, err)
	return stored
}

// memIdem is an in-memory idempotency.Store. Service tests use this instead of
// the Redis adapter; the Redis namespace regression test starts its own container.
type memIdem struct {
	mu   sync.Mutex
	recs map[string]idempotency.Record
}

func newMemIdem() *memIdem {
	return &memIdem{recs: make(map[string]idempotency.Record)}
}

func (m *memIdem) Get(_ context.Context, key string) (idempotency.Record, bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	rec, ok := m.recs[key]
	return rec, ok, nil
}

func (m *memIdem) Reserve(_ context.Context, key, fingerprint string, _ time.Duration) (bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.recs[key]; ok {
		return false, nil
	}
	m.recs[key] = idempotency.Record{Fingerprint: fingerprint}
	return true, nil
}

func (m *memIdem) Complete(_ context.Context, key, fingerprint, resourceID string, _ time.Duration) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.recs[key] = idempotency.Record{Fingerprint: fingerprint, ResourceID: resourceID}
	return nil
}

func (m *memIdem) Release(_ context.Context, key, _ string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.recs, key)
	return nil
}
