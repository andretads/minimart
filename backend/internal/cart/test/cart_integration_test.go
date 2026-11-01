//go:build integration

// Package test holds the cart domain's integration tests. Unlike the
// unit tests, these run against a REAL MongoDB and prove the adapter wiring
// the unit tests deliberately fake out with the in-memory repository.
//
// The database is provided by testcontainers-go: TestMain (main_test.go) starts
// an ephemeral MongoDB and tears it down afterwards. The adapter creates its
// own indexes in EnsureIndexes, so - unlike the SQL domains - there are no
// migrations to apply. They are excluded from a normal `go test ./...` by the
// build tag above; run them with:
//
//	go test -count=1 -tags=integration ./internal/cart/test/
package test

import (
	"context"
	"errors"
	"fmt"
	"minimart/internal/cart/application"
	"minimart/internal/cart/domain"
	"minimart/pkg/metrics"
	"strings"
	"sync"
	"testing"
	"time"

	addressclient "minimart/internal/address/presentation/module/client"
	cartinfra "minimart/internal/cart/infrastructure/mongo"
	catalogclient "minimart/internal/catalog/presentation/module/client"
	categoryclient "minimart/internal/category/presentation/module/client"
	customerclient "minimart/internal/customer/presentation/module/client"
	orderclient "minimart/internal/order/presentation/module/client"
	shippingclient "minimart/internal/shipping/presentation/module/client"

	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newTestRepo connects to the ephemeral MongoDB, ensures the adapter's indexes,
// and returns a repository over a fresh collection-carrying database handle.
// Each test uses distinct ids and user ids, so tests never interfere.
func newTestRepo(t *testing.T) *cartinfra.CartRepository {
	t.Helper()
	client, err := mongo.Connect(options.Client().ApplyURI(testMongoURI))
	require.NoError(t, err)
	t.Cleanup(func() { _ = client.Disconnect(t.Context()) })

	repo := cartinfra.NewCartRepository(client.Database("minimart"))
	require.NoError(t, repo.EnsureIndexes(t.Context()))
	return repo
}

// mustCart creates a valid cart for testing. Prefixes IDs with test name for isolation.
func mustCart(t *testing.T, id, customerID string) domain.Cart {
	t.Helper()
	cart, err := domain.NewCart(t.Name()+"-"+id, t.Name()+"-"+customerID)
	require.NoError(t, err)
	return cart
}

// mustItem creates a valid cart item for testing.
func mustItem(productID string, qty int) domain.Item {
	return domain.Item{
		ProductID:              productID,
		ProductNameSnapshot:    "Product " + productID,
		UnitPriceSnapshotCents: 1000,
		Quantity:               qty,
		CreatedAt:              time.Now().UTC(),
		UpdatedAt:              time.Now().UTC(),
	}
}

// TestCartRepository_GetByCustomerID_Create round-trips a cart through the real
// MongoDB adapter: GetOrCreate (which calls Create), then GetByCustomerID, and
// asserts the value comes back unchanged.
func TestCartRepository_GetByCustomerID_Create(t *testing.T) {
	ctx := t.Context()
	repo := newTestRepo(t)

	cart := mustCart(t, "it-cart-1", "it-cust-1")
	require.NoError(t, repo.Create(ctx, cart))

	got, err := repo.GetByCustomerID(ctx, cart.CustomerID)
	require.NoError(t, err)
	assert.Equal(t, cart.ID, got.ID)
	assert.Equal(t, cart.CustomerID, got.CustomerID)
	assert.Equal(t, cart.Status, got.Status)
	assert.Equal(t, cart.Version, got.Version)
	assert.Empty(t, got.Items)
}

// TestCartRepository_GetByCustomerID_NotFound returns ErrCartNotFound for missing customer.
func TestCartRepository_GetByCustomerID_NotFound(t *testing.T) {
	repo := newTestRepo(t)

	_, err := repo.GetByCustomerID(t.Context(), "it-missing-cust")
	assert.ErrorIs(t, err, domain.ErrCartNotFound)
}

// TestCartRepository_UniqueActiveCart enforces exactly one active cart per customer
// via the unique partial index on (customer_id, status) where status in (active, checkout_pending).
func TestCartRepository_UniqueActiveCart(t *testing.T) {
	ctx := t.Context()
	repo := newTestRepo(t)

	cart1 := mustCart(t, "it-cart-1", "it-cust-unique")
	require.NoError(t, repo.Create(ctx, cart1))

	cart2 := mustCart(t, "it-cart-2", "it-cust-unique")
	err := repo.Create(ctx, cart2)
	require.ErrorIs(t, err, domain.ErrCartAlreadyExists)
}

// TestCartRepository_Update_OptimisticLock proves CAS on version works correctly.
func TestCartRepository_Update_OptimisticLock(t *testing.T) {
	ctx := t.Context()
	repo := newTestRepo(t)

	cart := mustCart(t, "it-cart-opt", "it-cust-opt")
	require.NoError(t, repo.Create(ctx, cart))

	// Valid update with correct version
	cart.Items = append(cart.Items, mustItem("prod-1", 2))
	require.NoError(t, repo.Update(ctx, cart, 1))
	// Version is incremented in the repo, fetch to verify
	updated, err := repo.GetByCustomerID(ctx, cart.CustomerID)
	require.NoError(t, err)
	assert.Equal(t, 2, updated.Version)

	// Stale version should fail
	staleCart := updated
	staleCart.Items = append(staleCart.Items, mustItem("prod-2", 1))
	updErr := repo.Update(ctx, staleCart, 1) // version 1 is stale, current is 2
	require.ErrorIs(t, updErr, domain.ErrCartOptimisticLock)

	// Missing cart should fail with ErrCartNotFound
	missingCart := mustCart(t, "missing", "cust-missing")
	err = repo.Update(ctx, missingCart, 2)
	assert.ErrorIs(t, err, domain.ErrCartNotFound)
}

// TestCartRepository_Update_Concurrent hammers Update from parallel goroutines
// for the same cart - proving the CAS semantics under concurrency.
func TestCartRepository_Update_Concurrent(t *testing.T) {
	ctx := t.Context()
	repo := newTestRepo(t)

	cart := mustCart(t, "it-cart-concurrent", "it-cust-concurrent")
	require.NoError(t, repo.Create(ctx, cart))

	const writers = 10
	errs := make(chan error, writers)
	var wg sync.WaitGroup
	products := []string{"prod-a", "prod-b", "prod-c", "prod-d", "prod-e", "prod-f", "prod-g", "prod-h", "prod-i", "prod-j"}
	for i := range writers {
		wg.Go(func() {
			// Each goroutine tries to add a different item
			c := cart
			c.Items = append(c.Items, mustItem(products[i], 1))
			errs <- repo.Update(ctx, c, c.Version)
		})
	}
	wg.Wait()
	close(errs)

	// Exactly one should succeed (the first to CAS), others fail with OptimisticLock
	successCount := 0
	for err := range errs {
		if err == nil {
			successCount++
		} else {
			require.ErrorIs(t, err, domain.ErrCartOptimisticLock)
		}
	}
	assert.Equal(t, 1, successCount, "exactly one concurrent update should succeed")
}

// TestCartRepository_StatusTransitions tests the cart status transitions
// via Update: active -> checkout_pending -> consumed.
func TestCartRepository_StatusTransitions(t *testing.T) {
	ctx := t.Context()
	repo := newTestRepo(t)

	// Create active cart
	cart := mustCart(t, "it-cart-status", "it-cust-status")
	require.NoError(t, repo.Create(ctx, cart))
	assert.Equal(t, domain.StatusActive, cart.Status)

	// Transition to checkout_pending
	cart.Status = domain.StatusCheckoutPending
	cart.CheckoutKey = "idem-key-1"
	require.NoError(t, repo.Update(ctx, cart, 1))
	assert.Equal(t, domain.StatusCheckoutPending, cart.Status)

	// Transition to consumed
	cart.Status = domain.StatusConsumed
	cart.OrderID = "order-1"
	cart.CheckoutKey = ""
	require.NoError(t, repo.Update(ctx, cart, 2))
	assert.Equal(t, domain.StatusConsumed, cart.Status)
	assert.Equal(t, "order-1", cart.OrderID)
}

// TestCartRepository_GetByCustomerID_OnlyReturnsActiveOrPending verifies that
// GetByCustomerID only returns carts with status active or checkout_pending,
// not consumed carts.
func TestCartRepository_GetByCustomerID_OnlyReturnsActiveOrPending(t *testing.T) {
	ctx := t.Context()
	repo := newTestRepo(t)

	// Create and consume a cart
	cart := mustCart(t, "it-cart-consumed", "it-cust-get")
	require.NoError(t, repo.Create(ctx, cart))
	cart.Status = domain.StatusConsumed
	cart.OrderID = "order-1"
	require.NoError(t, repo.Update(ctx, cart, 1))

	// Should not find consumed cart
	_, getErr := repo.GetByCustomerID(ctx, cart.CustomerID)
	require.ErrorIs(t, getErr, domain.ErrCartNotFound)

	// But active cart should be found
	cart2 := mustCart(t, "it-cart-active", "it-cust-get")
	require.NoError(t, repo.Create(ctx, cart2))
	got, err := repo.GetByCustomerID(ctx, cart2.CustomerID)
	require.NoError(t, err)
	assert.Equal(t, cart2.ID, got.ID)
}

// newTestService creates a cart application Service with real MongoDB repo
// and fake contract implementations for the other domains. It also returns
// the order fake so tests can inject checkout failures.
func newTestService(t *testing.T) (*application.Service, *fakeOrder) {
	t.Helper()
	repo := newTestRepo(t)

	// Fake catalog
	catalog := &fakeCatalog{products: map[string]catalogclient.Product{
		"prod-1": {ID: "prod-1", Name: "Product 1", PriceCents: 1000},
		"prod-2": {ID: "prod-2", Name: "Product 2", PriceCents: 2000},
	}, categories: map[string]categoryclient.CategoryDTO{
		"cat-1": {ID: "cat-1", Name: "Test Category"},
	}}
	// Fake customer
	customer := newFakeCustomer()
	// Fake order
	order := newFakeOrder()
	// Fake category
	category := &fakeCategory{categories: map[string]categoryclient.CategoryDTO{
		"cat-1": {ID: "cat-1", Name: "Test Category"},
	}}

	mod := &testModules{fakeCatalog: catalog, fakeCustomer: customer, fakeOrder: order, fakeCategory: category, fakeShipping: &fakeShipping{}}
	return application.NewService(repo, mod, metrics.Noop{}), order
}

// Integration test helpers for contract fakes
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

type fakeCustomer struct {
	customers map[string]customerclient.Customer
	err       error
}

func newFakeCustomer() *fakeCustomer {
	return &fakeCustomer{customers: make(map[string]customerclient.Customer)}
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

type fakeOrder struct {
	orders map[string]orderclient.Order
	err    error
}

func newFakeOrder() *fakeOrder {
	return &fakeOrder{orders: make(map[string]orderclient.Order)}
}

func (f *fakeOrder) PlaceOrder(ctx context.Context, req orderclient.PlaceOrderRequest) (orderclient.Order, error) {
	if f.err != nil {
		return orderclient.Order{}, f.err
	}
	ord := orderclient.Order{
		ID:         "ord-" + req.IdempotencyKey,
		UserID:     req.UserID,
		CustomerID: "cust-" + req.UserID,
		Status:     "paid",
		TotalCents: 1000,
	}
	f.orders[req.IdempotencyKey] = ord
	return ord, nil
}

func (f *fakeOrder) GetOrder(_ context.Context, id string) (orderclient.Order, error) {
	return orderclient.Order{}, nil
}

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

type fakeShipping struct {
	shippingclient.ShippingClient
}

func (f *fakeShipping) GetQuote(_ context.Context, req shippingclient.GetQuoteRequest) (shippingclient.ShippingQuote, error) {
	return shippingclient.ShippingQuote{
		ID:                  req.ID,
		CustomerID:          "cust-" + strings.TrimPrefix(req.ID, "quote-"),
		MethodID:            "method-1",
		DestinationSnapshot: shippingclient.DestinationSnapshot{FullName: "Test User", Street: "1 Main St", City: "Testville", State: "TS", Zip: "12345", Country: "BR"},
		Status:              "active",
	}, nil
}

func (f *fakeShipping) ValidateQuote(_ context.Context, req shippingclient.ValidateQuoteRequest) (shippingclient.ShippingQuote, error) {
	return shippingclient.ShippingQuote{ID: req.ID, CustomerID: req.CustomerID, MethodID: "method-1", Status: "active"}, nil
}

func (f *fakeShipping) PriceQuote(_ context.Context, req shippingclient.PriceQuoteRequest) (shippingclient.ShippingQuote, error) {
	return shippingclient.ShippingQuote{
		ID:                  req.ID,
		CustomerID:          req.CustomerID,
		MethodID:            "method-1",
		ShippingCostCents:   0,
		SubtotalCents:       req.SubtotalCents,
		DestinationSnapshot: shippingclient.DestinationSnapshot{FullName: "Test User", Street: "1 Main St", City: "Testville", State: "TS", Zip: "12345", Country: "BR"},
		Status:              "active",
	}, nil
}

func (f *fakeShipping) GetMethod(_ context.Context, id string) (shippingclient.ShippingMethod, error) {
	return shippingclient.ShippingMethod{ID: id, Name: "Standard", Active: true}, nil
}

type testModules struct {
	*fakeCatalog
	*fakeCustomer
	*fakeOrder
	*fakeCategory
	*fakeShipping
	addressclient.AddressClient
}

// TestCartApplication_GetOrCreateCart_CreatesCart tests the full application layer
// with real MongoDB and fake contracts.
func TestCartApplication_GetOrCreateCart_CreatesCart(t *testing.T) {
	svc, _ := newTestService(t)

	cart, err := svc.GetOrCreateCart(t.Context(), application.GetOrCreateCartRequest{UserID: "it-user-create"})
	require.NoError(t, err)
	assert.Equal(t, "cust-it-user-create", cart.CustomerID)
	assert.Equal(t, domain.StatusActive, cart.Status)
	assert.NotEmpty(t, cart.ID)
}

// TestCartApplication_GetOrCreateCart_Idempotent verifies that calling GetOrCreateCart
// twice for the same user returns the same cart.
func TestCartApplication_GetOrCreateCart_Idempotent(t *testing.T) {
	svc, _ := newTestService(t)

	cart1, err := svc.GetOrCreateCart(t.Context(), application.GetOrCreateCartRequest{UserID: "it-user-idem"})
	require.NoError(t, err)

	cart2, err := svc.GetOrCreateCart(t.Context(), application.GetOrCreateCartRequest{UserID: "it-user-idem"})
	require.NoError(t, err)

	assert.Equal(t, cart1.ID, cart2.ID)
}

// TestCartApplication_AddItem_AddsAndIncrements tests adding items with real MongoDB.
func TestCartApplication_AddItem_AddsAndIncrements(t *testing.T) {
	svc, _ := newTestService(t)

	// Add first item
	cart, err := svc.AddItem(t.Context(), application.AddItemRequest{UserID: "it-user-add", ProductID: "prod-1", Quantity: 2})
	require.NoError(t, err)
	require.Len(t, cart.Items, 1)
	assert.Equal(t, "prod-1", cart.Items[0].ProductID)
	assert.Equal(t, 2, cart.Items[0].Quantity)

	// Increment existing item
	cart, err = svc.AddItem(t.Context(), application.AddItemRequest{UserID: "it-user-add", ProductID: "prod-1", Quantity: 3})
	require.NoError(t, err)
	require.Len(t, cart.Items, 1)
	assert.Equal(t, 5, cart.Items[0].Quantity)

	// Add different product
	cart, err = svc.AddItem(t.Context(), application.AddItemRequest{UserID: "it-user-add", ProductID: "prod-2", Quantity: 1})
	require.NoError(t, err)
	require.Len(t, cart.Items, 2)
}

// TestCartApplication_Checkout_FullFlow tests the complete checkout flow
// with real MongoDB and fake contracts.
func TestCartApplication_Checkout_FullFlow(t *testing.T) {
	repo := newTestRepo(t)
	svc, _ := newTestService(t)

	// Add items to cart
	_, err := svc.AddItem(t.Context(), application.AddItemRequest{UserID: "it-user-checkout", ProductID: "prod-1", Quantity: 2})
	require.NoError(t, err)

	// Checkout
	ord, err := svc.CheckoutCart(t.Context(), application.CheckoutCartRequest{UserID: "it-user-checkout", IdempotencyKey: "idem-checkout-1", ShippingQuoteID: "quote-it-user-checkout", AddressID: "addr-1"})
	require.NoError(t, err)
	assert.Equal(t, "ord-idem-checkout-1", ord.ID)
	assert.Equal(t, "paid", ord.Status)
	// The order belongs to the user, and the customer record is resolved from it.
	assert.Equal(t, "it-user-checkout", ord.UserID)
	assert.Equal(t, "cust-it-user-checkout", ord.CustomerID)

	// Verify cart is consumed by fetching directly from repo (consumed carts are not returned by GetByCustomerID)
	_, err = repo.GetByCustomerID(t.Context(), "cust-it-user-checkout")
	require.ErrorIs(t, err, domain.ErrCartNotFound, "consumed cart should not be returned by GetByCustomerID")

	// But we can verify the cart was consumed by checking the order was created
	assert.Equal(t, "ord-idem-checkout-1", ord.ID)
	assert.Equal(t, "paid", ord.Status)
}

// TestCartApplication_Checkout_EmptyCartFails verifies checkout fails on empty cart.
func TestCartApplication_Checkout_EmptyCartFails(t *testing.T) {
	svc, _ := newTestService(t)

	_, err := svc.CheckoutCart(t.Context(), application.CheckoutCartRequest{UserID: "it-user-empty", IdempotencyKey: "idem-empty", ShippingQuoteID: "quote-it-user-empty", AddressID: "addr-1"})
	require.ErrorIs(t, err, domain.ErrCartEmpty)
}

// TestCartApplication_Checkout_IdempotentRetry verifies retry with same
// idempotency key after successful checkout returns the same order.
func TestCartApplication_Checkout_IdempotentRetry(t *testing.T) {
	svc, _ := newTestService(t)

	_, err := svc.AddItem(t.Context(), application.AddItemRequest{UserID: "it-user-retry", ProductID: "prod-1", Quantity: 2})
	require.NoError(t, err)

	// First checkout
	ord1, err := svc.CheckoutCart(t.Context(), application.CheckoutCartRequest{UserID: "it-user-retry", IdempotencyKey: "idem-retry", ShippingQuoteID: "quote-it-user-retry", AddressID: "addr-1"})
	require.NoError(t, err)

	// Retry with same key
	ord2, err := svc.CheckoutCart(t.Context(), application.CheckoutCartRequest{UserID: "it-user-retry", IdempotencyKey: "idem-retry", ShippingQuoteID: "quote-it-user-retry", AddressID: "addr-1"})
	require.NoError(t, err)
	assert.Equal(t, ord1.ID, ord2.ID)
}

// TestCartApplication_Checkout_OrderFailurePreservesCart verifies that when
// order creation fails definitively, the cart is returned to active state.
func TestCartApplication_Checkout_OrderFailurePreservesCart(t *testing.T) {
	svc, orderFake := newTestService(t)

	_, err := svc.AddItem(t.Context(), application.AddItemRequest{UserID: "it-user-fail", ProductID: "prod-1", Quantity: 2})
	require.NoError(t, err)

	// Inject order failure
	orderFake.err = orderclient.ErrInsufficientStock

	_, err = svc.CheckoutCart(t.Context(), application.CheckoutCartRequest{UserID: "it-user-fail", IdempotencyKey: "idem-fail", ShippingQuoteID: "quote-it-user-fail", AddressID: "addr-1"})
	require.ErrorIs(t, err, orderclient.ErrInsufficientStock)

	// Cart should be back to active
	cart, err := svc.GetCart(t.Context(), application.GetCartRequest{UserID: "it-user-fail"})
	require.NoError(t, err)
	assert.Equal(t, domain.StatusActive, cart.Status)
}

// TestCartApplication_Checkout_ConcurrentCheckout tests that concurrent
// checkout attempts are serialized - only one succeeds.
func TestCartApplication_Checkout_ConcurrentCheckout(t *testing.T) {
	repo := newTestRepo(t)
	catalog := &fakeCatalog{products: map[string]catalogclient.Product{
		"prod-1": {ID: "prod-1", Name: "Product 1", PriceCents: 1000},
	}, categories: map[string]categoryclient.CategoryDTO{
		"cat-1": {ID: "cat-1", Name: "Test Category"},
	}}
	customer := newFakeCustomer()
	order := newFakeOrder()
	category := &fakeCategory{categories: map[string]categoryclient.CategoryDTO{
		"cat-1": {ID: "cat-1", Name: "Test Category"},
	}}
	mod := &testModules{fakeCatalog: catalog, fakeCustomer: customer, fakeOrder: order, fakeCategory: category, fakeShipping: &fakeShipping{}}
	svc := application.NewService(repo, mod, metrics.Noop{})

	_, err := svc.AddItem(t.Context(), application.AddItemRequest{UserID: "it-user-concurrent", ProductID: "prod-1", Quantity: 2})
	require.NoError(t, err)

	const writers = 10
	errs := make(chan error, writers)
	orders := make(chan orderclient.Order, writers)
	var wg sync.WaitGroup
	for i := range writers {
		wg.Go(func() {
			idem := fmt.Sprintf("idem-concurrent-%d", i)
			ord, err := svc.CheckoutCart(t.Context(), application.CheckoutCartRequest{UserID: "it-user-concurrent", IdempotencyKey: idem, ShippingQuoteID: "quote-it-user-concurrent", AddressID: "addr-1"})
			errs <- err
			if err == nil {
				orders <- ord
			}
		})
	}
	wg.Wait()
	close(errs)
	close(orders)

	// Exactly one should succeed; the rest lose either the optimistic-lock race
	// or observe the winner's checkout already in progress.
	successCount := 0
	for err := range errs {
		if err == nil {
			successCount++
			continue
		}
		require.True(t,
			errors.Is(err, domain.ErrCartOptimisticLock) || errors.Is(err, application.ErrCartCheckoutInProgress),
			"unexpected checkout error: %v", err,
		)
	}
	assert.Equal(t, 1, successCount, "exactly one concurrent checkout should succeed")

	// The successful order should have a valid ID
	ord := <-orders
	require.NotEmpty(t, ord.ID)
}
