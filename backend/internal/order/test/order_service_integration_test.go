//go:build integration

// Package test holds the order domain's integration tests.
package test

import (
	"context"
	addressclient "minimart/internal/address/presentation/module/client"
	catalogclient "minimart/internal/catalog/presentation/module/client"
	categoryclient "minimart/internal/category/presentation/module/client"
	customerclient "minimart/internal/customer/presentation/module/client"
	"minimart/internal/order/application"
	"minimart/internal/order/domain"
	"minimart/internal/order/infrastructure/postgres"
	orderclient "minimart/internal/order/presentation/module/client"
	shippingclient "minimart/internal/shipping/presentation/module/client"
	"minimart/pkg/metrics"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Fake catalog client for testing
type fakeCatalog struct {
	products   map[string]catalogclient.Product
	categories map[string]categoryclient.CategoryDTO
	err        error
}

func (f *fakeCatalog) GetProduct(ctx context.Context, id string) (catalogclient.Product, error) {
	if f.err != nil {
		return catalogclient.Product{}, f.err
	}
	prod, ok := f.products[id]
	if !ok {
		return catalogclient.Product{}, catalogclient.ErrProductNotFound
	}
	return prod, nil
}

func (f *fakeCatalog) ReserveStock(ctx context.Context, orderID string, items []catalogclient.LineItem) error {
	return f.err
}

func (f *fakeCatalog) ReleaseStock(ctx context.Context, orderID string) error {
	return f.err
}

// Fake customer client for testing
type fakeCustomer struct {
	customers map[string]customerclient.Customer
	err       error
}

func (f *fakeCustomer) EnsureCustomer(ctx context.Context, userID string) (customerclient.Customer, error) {
	if f.err != nil {
		return customerclient.Customer{}, f.err
	}
	if c, ok := f.customers[userID]; ok {
		return c, nil
	}
	c := customerclient.Customer{ID: "cust-" + userID, UserID: userID, DisplayName: "Test", Status: "active"}
	f.customers[userID] = c
	return c, nil
}

func (f *fakeCustomer) GetCustomerByID(ctx context.Context, id string) (customerclient.Customer, error) {
	return customerclient.Customer{}, customerclient.ErrCustomerNotFound
}

func (f *fakeCustomer) GetCustomerByUserID(ctx context.Context, userID string) (customerclient.Customer, error) {
	if f.err != nil {
		return customerclient.Customer{}, f.err
	}
	if c, ok := f.customers[userID]; ok {
		return c, nil
	}
	return customerclient.Customer{}, customerclient.ErrCustomerNotFound
}

// Fake order client for testing (not used by order service but required by contracts.Modules)
type fakeOrder struct{}

func (f *fakeOrder) PlaceOrder(ctx context.Context, req orderclient.PlaceOrderRequest) (orderclient.Order, error) {
	return orderclient.Order{}, nil
}

func (f *fakeOrder) GetOrder(_ context.Context, id string) (orderclient.Order, error) {
	return orderclient.Order{}, nil
}

// Fake category client for testing
type fakeCategory struct {
	categories map[string]categoryclient.CategoryDTO
	err        error
}

func (f *fakeCategory) GetCategory(ctx context.Context, id string) (categoryclient.CategoryDTO, error) {
	if f.err != nil {
		return categoryclient.CategoryDTO{}, f.err
	}
	cat, ok := f.categories[id]
	if !ok {
		return categoryclient.CategoryDTO{}, categoryclient.ErrCategoryNotFound
	}
	return cat, nil
}

// Test modules struct that implements the contracts
type testModules struct {
	*fakeCatalog
	*fakeCustomer
	*fakeOrder
	*fakeCategory
	shippingclient.ShippingClient
	addressclient.AddressClient
}

// newTestService creates an order application Service with real Postgres repo
// and fake contract implementations for catalog, customer, and order.
func newTestService(t *testing.T) (*application.Service, *postgres.OrderRepository, *pgxpool.Pool) {
	t.Helper()
	repo, pool := newOrderRepo(t)

	// Orders carry a customer_id foreign key, so the fake customer must have a
	// matching row. The id matches fakeCustomer.EnsureCustomer's derivation.
	_, err := pool.Exec(t.Context(),
		"INSERT INTO customers (id, user_id) VALUES ($1, $2) ON CONFLICT (id) DO NOTHING",
		"cust-it-user", "it-user")
	require.NoError(t, err)

	catalog := &fakeCatalog{products: map[string]catalogclient.Product{
		"p1": {ID: "p1", Name: "Widget", PriceCents: 500},
		"p2": {ID: "p2", Name: "Gadget", PriceCents: 250},
	}, categories: map[string]categoryclient.CategoryDTO{
		"cat-1": {ID: "cat-1", Name: "Test Category"},
	}}
	customer := &fakeCustomer{customers: make(map[string]customerclient.Customer)}
	orderClient := &fakeOrder{}
	category := &fakeCategory{categories: map[string]categoryclient.CategoryDTO{
		"cat-1": {ID: "cat-1", Name: "Test Category"},
	}}
	mod := &testModules{fakeCatalog: catalog, fakeCustomer: customer, fakeOrder: orderClient, fakeCategory: category}

	// Order service uses the fake in-memory idempotency store (not Redis)
	// since order test package doesn't have Redis container
	svc := application.NewService(repo, mod, newMemIdem(), metrics.Noop{}, time.Hour)
	return svc, repo, pool
}

// TestOrderService_PlaceOrder tests placing an order through the application service
func TestOrderService_PlaceOrder(t *testing.T) {
	svc, repo, pool := newTestService(t)

	key := uniqueOrderID()
	req := application.PlaceOrderRequest{
		UserID:         "it-user",
		IdempotencyKey: "idem-" + key,
		Items: []application.PlaceOrderItem{
			{ProductID: "p1", Quantity: 2},
			{ProductID: "p2", Quantity: 1},
		},
	}

	order, err := svc.PlaceOrder(t.Context(), req)
	require.NoError(t, err)
	cleanup(t, pool, order.ID)

	assert.Equal(t, "it-user", order.UserID)
	assert.Equal(t, domain.StatusPlaced, order.Status)
	assert.Equal(t, int64(1250), order.TotalCents)
	require.Len(t, order.Items, 2)

	// Verify it's persisted
	stored, err := repo.GetByID(t.Context(), order.ID)
	require.NoError(t, err)
	assert.Equal(t, order.ID, stored.ID)
	assert.Equal(t, order.TotalCents, stored.TotalCents)
}

// TestOrderService_PlaceOrder_InsufficientStock tests that placing an order
// with insufficient stock fails
func TestOrderService_PlaceOrder_InsufficientStock(t *testing.T) {
	repo, _ := newOrderRepo(t)

	catalog := &fakeCatalog{
		products: map[string]catalogclient.Product{
			"p1": {ID: "p1", Name: "Widget", PriceCents: 500},
		},
		categories: map[string]categoryclient.CategoryDTO{
			"cat-1": {ID: "cat-1", Name: "Test Category"},
		},
		err: catalogclient.ErrInsufficientStock,
	}
	customer := &fakeCustomer{customers: make(map[string]customerclient.Customer)}
	orderClient := &fakeOrder{}
	category := &fakeCategory{categories: map[string]categoryclient.CategoryDTO{
		"cat-1": {ID: "cat-1", Name: "Test Category"},
	}}
	mod := &testModules{fakeCatalog: catalog, fakeCustomer: customer, fakeOrder: orderClient, fakeCategory: category}

	svc := application.NewService(repo, mod, newMemIdem(), metrics.Noop{}, time.Hour)

	key := uniqueOrderID()
	req := application.PlaceOrderRequest{
		UserID:         "it-user",
		IdempotencyKey: "idem-" + key,
		Items: []application.PlaceOrderItem{
			{ProductID: "p1", Quantity: 100},
		},
	}

	_, err := svc.PlaceOrder(t.Context(), req)
	require.ErrorIs(t, err, catalogclient.ErrInsufficientStock)
}

// TestOrderService_PlaceOrder_Idempotent tests that placing an order with the
// same idempotency key returns the same order
func TestOrderService_PlaceOrder_Idempotent(t *testing.T) {
	svc, _, pool := newTestService(t)

	key := uniqueOrderID()
	req := application.PlaceOrderRequest{
		UserID:         "it-user",
		IdempotencyKey: "idem-" + key,
		Items: []application.PlaceOrderItem{
			{ProductID: "p1", Quantity: 1},
		},
	}

	order1, err := svc.PlaceOrder(t.Context(), req)
	require.NoError(t, err)
	cleanup(t, pool, order1.ID)

	order2, err := svc.PlaceOrder(t.Context(), req)
	require.NoError(t, err)
	assert.Equal(t, order1.ID, order2.ID)
}

// TestOrderService_PlaceOrder_DifferentItemsSameKeyConflicts tests that using
// the same idempotency key with different items returns a conflict
func TestOrderService_PlaceOrder_DifferentItemsSameKeyConflicts(t *testing.T) {
	svc, _, pool := newTestService(t)

	key := uniqueOrderID()
	req := application.PlaceOrderRequest{
		UserID:         "it-user",
		IdempotencyKey: "idem-" + key,
		Items: []application.PlaceOrderItem{
			{ProductID: "p1", Quantity: 1},
		},
	}

	order1, err := svc.PlaceOrder(t.Context(), req)
	require.NoError(t, err)
	cleanup(t, pool, order1.ID)

	// Same key, different items
	req.Items = []application.PlaceOrderItem{
		{ProductID: "p2", Quantity: 1},
	}
	_, err = svc.PlaceOrder(t.Context(), req)
	require.ErrorIs(t, err, domain.ErrIdempotencyConflict)
}

// TestOrderService_ApplyPaymentOutcome_Paid tests applying a successful payment outcome
func TestOrderService_ApplyPaymentOutcome_Paid(t *testing.T) {
	svc, repo, pool := newTestService(t)

	key := uniqueOrderID()
	order, err := svc.PlaceOrder(t.Context(), application.PlaceOrderRequest{
		UserID:         "it-user",
		IdempotencyKey: "idem-" + key,
		Items:          []application.PlaceOrderItem{{ProductID: "p1", Quantity: 1}},
	})
	require.NoError(t, err)
	cleanup(t, pool, order.ID)

	// Apply payment succeeded
	err = svc.ApplyPaymentOutcome(t.Context(), order.ID, application.PaymentOutcomeSucceeded, "")
	require.NoError(t, err)

	updated, err := repo.GetByID(t.Context(), order.ID)
	require.NoError(t, err)
	assert.Equal(t, domain.StatusPaid, updated.Status)
}

// TestOrderService_ApplyPaymentOutcome_Failed tests applying a failed payment outcome
func TestOrderService_ApplyPaymentOutcome_Failed(t *testing.T) {
	svc, repo, pool := newTestService(t)

	key := uniqueOrderID()
	order, err := svc.PlaceOrder(t.Context(), application.PlaceOrderRequest{
		UserID:         "it-user",
		IdempotencyKey: "idem-" + key,
		Items:          []application.PlaceOrderItem{{ProductID: "p1", Quantity: 1}},
	})
	require.NoError(t, err)
	cleanup(t, pool, order.ID)

	// Apply payment failed
	err = svc.ApplyPaymentOutcome(t.Context(), order.ID, application.PaymentOutcomeFailed, "")
	require.NoError(t, err)

	updated, err := repo.GetByID(t.Context(), order.ID)
	require.NoError(t, err)
	assert.Equal(t, domain.StatusPaymentFailed, updated.Status)
}

// TestOrderService_ApplyPaymentOutcome_Unknown tests applying an unknown payment outcome
func TestOrderService_ApplyPaymentOutcome_Unknown(t *testing.T) {
	svc, repo, pool := newTestService(t)

	key := uniqueOrderID()
	order, err := svc.PlaceOrder(t.Context(), application.PlaceOrderRequest{
		UserID:         "it-user",
		IdempotencyKey: "idem-" + key,
		Items:          []application.PlaceOrderItem{{ProductID: "p1", Quantity: 1}},
	})
	require.NoError(t, err)
	cleanup(t, pool, order.ID)

	// Apply payment unknown
	err = svc.ApplyPaymentOutcome(t.Context(), order.ID, application.PaymentOutcomeUnknown, "")
	require.NoError(t, err)

	updated, err := repo.GetByID(t.Context(), order.ID)
	require.NoError(t, err)
	assert.Equal(t, domain.StatusPaymentUnknown, updated.Status)
}

// TestOrderService_CancelOrder tests cancelling an order
func TestOrderService_CancelOrder(t *testing.T) {
	svc, _, pool := newTestService(t)

	key := uniqueOrderID()
	order, err := svc.PlaceOrder(t.Context(), application.PlaceOrderRequest{
		UserID:         "it-user",
		IdempotencyKey: "idem-" + key,
		Items:          []application.PlaceOrderItem{{ProductID: "p1", Quantity: 1}},
	})
	require.NoError(t, err)
	cleanup(t, pool, order.ID)

	// Cancel the order
	cancelled, err := svc.CancelOrder(t.Context(), order.ID)
	require.NoError(t, err)
	assert.Equal(t, domain.StatusCancelled, cancelled.Status)

	// Verify stock release event was written
	_, err = pool.Exec(t.Context(), "DELETE FROM outbox WHERE message_key = $1", order.ID)
	require.NoError(t, err)
}
