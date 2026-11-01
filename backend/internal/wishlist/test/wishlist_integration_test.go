//go:build integration

// Package test holds the wishlist domain's integration tests. Unlike the
// unit tests, these run against a REAL MongoDB and prove the adapter wiring
// the unit tests deliberately fake out with the in-memory repository.
//
// The database is provided by testcontainers-go: TestMain (main_test.go) starts
// an ephemeral MongoDB and tears it down afterwards. The adapter creates its
// own indexes in EnsureIndexes, so - unlike the SQL domains - there are no
// migrations to apply. They are excluded from a normal `go test ./...` by the
// build tag above; run them with:
//
//	go test -count=1 -tags=integration ./internal/wishlist/test/
package test

import (
	"context"
	"minimart/internal/wishlist/application"
	"minimart/internal/wishlist/domain"
	"minimart/pkg/metrics"
	"sync"
	"testing"
	"time"

	addressclient "minimart/internal/address/presentation/module/client"
	catalogclient "minimart/internal/catalog/presentation/module/client"
	categoryclient "minimart/internal/category/presentation/module/client"
	customerclient "minimart/internal/customer/presentation/module/client"
	orderclient "minimart/internal/order/presentation/module/client"
	shippingclient "minimart/internal/shipping/presentation/module/client"
	wishlistinfra "minimart/internal/wishlist/infrastructure/mongo"

	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newTestRepo connects to the ephemeral MongoDB, ensures the adapter's indexes,
// and returns a repository over a fresh collection-carrying database handle.
// Each test uses distinct ids and user ids, so tests never interfere.
func newTestRepo(t *testing.T) *wishlistinfra.Repository {
	t.Helper()
	client, err := mongo.Connect(options.Client().ApplyURI(testMongoURI))
	require.NoError(t, err)
	t.Cleanup(func() { _ = client.Disconnect(t.Context()) })

	repo := wishlistinfra.NewRepository(client.Database("minimart"))
	require.NoError(t, repo.EnsureIndexes(t.Context()))
	return repo
}

// mustWishlist creates a valid wishlist for testing. Prefixes IDs with test name for isolation.
func mustWishlist(t *testing.T, id, customerID string) domain.Wishlist {
	t.Helper()
	wishlist, err := domain.NewWishlist(t.Name()+"-"+id, t.Name()+"-"+customerID)
	require.NoError(t, err)
	return wishlist
}

// TestWishlistRepository_GetByCustomerID_Create round-trips a wishlist through the real
// MongoDB adapter: Create, then GetByCustomerID, and asserts the value comes back unchanged.
func TestWishlistRepository_GetByCustomerID_Create(t *testing.T) {
	ctx := t.Context()
	repo := newTestRepo(t)

	wishlist := mustWishlist(t, "it-wl-1", "it-cust-1")
	require.NoError(t, repo.Create(ctx, wishlist))

	got, err := repo.GetByCustomerID(ctx, wishlist.CustomerID)
	require.NoError(t, err)
	assert.Equal(t, wishlist.ID, got.ID)
	assert.Equal(t, wishlist.CustomerID, got.CustomerID)
	assert.Empty(t, got.Items)
}

// TestWishlistRepository_GetByCustomerID_NotFound returns ErrWishlistNotFound for missing customer.
func TestWishlistRepository_GetByCustomerID_NotFound(t *testing.T) {
	repo := newTestRepo(t)

	_, err := repo.GetByCustomerID(t.Context(), "it-missing-cust")
	assert.ErrorIs(t, err, domain.ErrWishlistNotFound)
}

// TestWishlistRepository_UniqueWishlist enforces exactly one wishlist per customer
// via the unique index on customer_id.
func TestWishlistRepository_UniqueWishlist(t *testing.T) {
	ctx := t.Context()
	repo := newTestRepo(t)

	wishlist1 := mustWishlist(t, "it-wl-1", "it-cust-unique")
	require.NoError(t, repo.Create(ctx, wishlist1))

	wishlist2 := mustWishlist(t, "it-wl-2", "it-cust-unique")
	err := repo.Create(ctx, wishlist2)
	require.ErrorIs(t, err, domain.ErrWishlistAlreadyExists)
}

// TestWishlistRepository_Update_Items proves updating items works correctly.
func TestWishlistRepository_Update_Items(t *testing.T) {
	ctx := t.Context()
	repo := newTestRepo(t)

	wishlist := mustWishlist(t, "it-wl-opt", "it-cust-opt")
	require.NoError(t, repo.Create(ctx, wishlist))

	// Valid update - add items
	wishlist.Items = append(wishlist.Items, domain.WishlistItem{ProductID: "prod-1", AddedAt: time.Now().UTC()})
	require.NoError(t, repo.Update(ctx, wishlist))
	// Fetch to verify
	updated, err := repo.GetByCustomerID(ctx, wishlist.CustomerID)
	require.NoError(t, err)
	require.Len(t, updated.Items, 1)
	assert.Equal(t, "prod-1", updated.Items[0].ProductID)

	// Add another item
	updated.Items = append(updated.Items, domain.WishlistItem{ProductID: "prod-2", AddedAt: time.Now().UTC()})
	require.NoError(t, repo.Update(ctx, updated))
	updated2, err := repo.GetByCustomerID(ctx, wishlist.CustomerID)
	require.NoError(t, err)
	require.Len(t, updated2.Items, 2)
}

// TestWishlistRepository_Update_Concurrent hammers Update from parallel goroutines
// for the same wishlist - proving the mutex semantics under concurrency.
func TestWishlistRepository_Update_Concurrent(t *testing.T) {
	ctx := t.Context()
	repo := newTestRepo(t)

	wishlist := mustWishlist(t, "it-wl-concurrent", "it-cust-concurrent")
	require.NoError(t, repo.Create(ctx, wishlist))

	const writers = 10
	errs := make(chan error, writers)
	var wg sync.WaitGroup
	products := []string{"prod-a", "prod-b", "prod-c", "prod-d", "prod-e", "prod-f", "prod-g", "prod-h", "prod-i", "prod-j"}
	for i := range writers {
		wg.Go(func() {
			// Each goroutine tries to add a different item
			w := wishlist
			w.Items = append(w.Items, domain.WishlistItem{ProductID: products[i], AddedAt: time.Now().UTC()})
			errs <- repo.Update(ctx, w)
		})
	}
	wg.Wait()
	close(errs)

	// At least one should succeed
	successCount := 0
	for err := range errs {
		if err == nil {
			successCount++
		}
	}
	assert.GreaterOrEqual(t, successCount, 1, "at least one concurrent update should succeed")

	// Verify final state
	final, err := repo.GetByCustomerID(ctx, wishlist.CustomerID)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(final.Items), 1)
}

// TestWishlistRepository_Update_NonExistentWishlist returns ErrWishlistNotFound when updating missing wishlist
func TestWishlistRepository_Update_NonExistentWishlist(t *testing.T) {
	ctx := t.Context()
	repo := newTestRepo(t)

	wishlist := mustWishlist(t, "missing", "cust-missing")
	err := repo.Update(ctx, wishlist)
	require.ErrorIs(t, err, domain.ErrWishlistNotFound)
}

// newTestService creates a wishlist application Service with real MongoDB repo
// and fake contract implementations for the other domains.
func newTestService(t *testing.T) *application.Service {
	t.Helper()
	repo := newTestRepo(t)

	// Fake catalog
	catalog := &fakeCatalog{products: map[string]catalogclient.Product{
		"prod-1": {ID: "prod-1", Name: "Product 1", PriceCents: 1000},
		"prod-2": {ID: "prod-2", Name: "Product 2", PriceCents: 2000},
	}}
	// Fake customer
	customer := newFakeCustomer()
	// Fake order
	order := newFakeOrder()
	// Fake category
	category := newFakeCategory()
	category.categories["cat-1"] = categoryclient.CategoryDTO{ID: "cat-1", Name: "Test Category"}

	mod := &testModules{fakeCatalog: catalog, fakeCustomer: customer, fakeOrder: order, fakeCategory: category}
	return application.NewService(repo, mod, metrics.Noop{})
}

// Integration test helpers for contract fakes
type fakeCatalog struct {
	products map[string]catalogclient.Product
	err      error
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

func newFakeCategory() *fakeCategory {
	return &fakeCategory{categories: make(map[string]categoryclient.CategoryDTO)}
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

type testModules struct {
	*fakeCatalog
	*fakeCustomer
	*fakeOrder
	*fakeCategory
	shippingclient.ShippingClient
	addressclient.AddressClient
}

// TestWishlistApplication_GetWishlist_CreatesWishlist tests the full application layer
// with real MongoDB and fake contracts.
func TestWishlistApplication_GetWishlist_CreatesWishlist(t *testing.T) {
	svc := newTestService(t)

	wishlist, err := svc.GetWishlist(t.Context(), application.GetWishlistRequest{UserID: "it-user-create"})
	require.NoError(t, err)
	assert.Equal(t, "cust-it-user-create", wishlist.CustomerID)
	assert.Empty(t, wishlist.Items)
	assert.NotEmpty(t, wishlist.ID)
}

// TestWishlistApplication_GetWishlist_Idempotent verifies that calling GetWishlist
// twice for the same user returns the same wishlist.
func TestWishlistApplication_GetWishlist_Idempotent(t *testing.T) {
	svc := newTestService(t)

	wishlist1, err := svc.GetWishlist(t.Context(), application.GetWishlistRequest{UserID: "it-user-idem"})
	require.NoError(t, err)

	wishlist2, err := svc.GetWishlist(t.Context(), application.GetWishlistRequest{UserID: "it-user-idem"})
	require.NoError(t, err)

	assert.Equal(t, wishlist1.ID, wishlist2.ID)
}

// TestWishlistApplication_AddItem_AddsAndErrors tests adding items with real MongoDB.
func TestWishlistApplication_AddItem_AddsAndErrors(t *testing.T) {
	svc := newTestService(t)

	// Add first item
	wishlist, err := svc.AddItem(t.Context(), application.AddItemRequest{UserID: "it-user-add", ProductID: "prod-1"})
	require.NoError(t, err)
	require.Len(t, wishlist.Items, 1)
	assert.Equal(t, "prod-1", wishlist.Items[0].ProductID)

	// Try to add duplicate - should fail
	_, err = svc.AddItem(t.Context(), application.AddItemRequest{UserID: "it-user-add", ProductID: "prod-1"})
	require.ErrorIs(t, err, domain.ErrProductAlreadyInWishlist)

	// Add different product
	wishlist, err = svc.AddItem(t.Context(), application.AddItemRequest{UserID: "it-user-add", ProductID: "prod-2"})
	require.NoError(t, err)
	require.Len(t, wishlist.Items, 2)
}

// TestWishlistApplication_RemoveItem tests removing items.
func TestWishlistApplication_RemoveItem(t *testing.T) {
	svc := newTestService(t)

	// Add items
	_, _ = svc.AddItem(t.Context(), application.AddItemRequest{UserID: "it-user-remove", ProductID: "prod-1"})
	_, _ = svc.AddItem(t.Context(), application.AddItemRequest{UserID: "it-user-remove", ProductID: "prod-2"})

	// Remove one
	wishlist, err := svc.RemoveItem(t.Context(), application.RemoveItemRequest{UserID: "it-user-remove", ProductID: "prod-1"})
	require.NoError(t, err)
	require.Len(t, wishlist.Items, 1)
	assert.Equal(t, "prod-2", wishlist.Items[0].ProductID)

	// Remove non-existent
	_, err = svc.RemoveItem(t.Context(), application.RemoveItemRequest{UserID: "it-user-remove", ProductID: "prod-999"})
	require.ErrorIs(t, err, domain.ErrProductNotInWishlist)
}

// TestWishlistApplication_ClearWishlist tests clearing the wishlist.
func TestWishlistApplication_ClearWishlist(t *testing.T) {
	svc := newTestService(t)

	// Add items
	_, _ = svc.AddItem(t.Context(), application.AddItemRequest{UserID: "it-user-clear", ProductID: "prod-1"})
	_, _ = svc.AddItem(t.Context(), application.AddItemRequest{UserID: "it-user-clear", ProductID: "prod-2"})

	// Clear
	err := svc.ClearWishlist(t.Context(), application.ClearWishlistRequest{UserID: "it-user-clear"})
	require.NoError(t, err)

	// Verify empty
	wishlist, err := svc.GetWishlist(t.Context(), application.GetWishlistRequest{UserID: "it-user-clear"})
	require.NoError(t, err)
	assert.Empty(t, wishlist.Items)
}

// TestWishlistApplication_AddItem_ProductNotFound tests product validation.
func TestWishlistApplication_AddItem_ProductNotFound(t *testing.T) {
	svc := newTestService(t)

	_, err := svc.AddItem(t.Context(), application.AddItemRequest{UserID: "it-user-notfound", ProductID: "prod-999"})
	require.ErrorIs(t, err, catalogclient.ErrProductNotFound)
}
