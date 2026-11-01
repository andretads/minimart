//go:build integration

package test

import (
	"context"
	"crypto/rand"
	"errors"
	"minimart/internal/cart/application"
	"minimart/internal/cart/domain"
	"minimart/pkg/metrics"
	"strings"
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

// The logical fixtures from features/cart/*.feature.
const (
	featureProductID         = "p1"
	featureProductName       = "Widget"
	featureProductPriceCents = int64(500)

	featureExtraProductID    = "p2"
	featureExtraProductName  = "Gadget"
	featureExtraProductPrice = int64(700)

	featureAddressID   = "addr-1"
	featureCheckoutKey = "k1"
)

// newTestRepo connects to the ephemeral MongoDB, ensures the adapter's indexes,
// and returns a repository over a fresh database handle. Each test uses distinct
// user ids, so tests never interfere.
func newTestRepo(t *testing.T) *cartinfra.CartRepository {
	t.Helper()
	client, err := mongo.Connect(options.Client().ApplyURI(testMongoURI))
	require.NoError(t, err)
	t.Cleanup(func() { _ = client.Disconnect(t.Context()) })

	repo := cartinfra.NewCartRepository(client.Database("minimart"))
	require.NoError(t, repo.EnsureIndexes(t.Context()))
	return repo
}

// uniqueUser gives every scenario its own user and customer namespace.
func uniqueUser() string {
	return "user-" + rand.Text()
}

type fakeCatalog struct {
	products map[string]catalogclient.Product
	err      error
}

func (f *fakeCatalog) GetProduct(_ context.Context, id string) (catalogclient.Product, error) {
	if f.err != nil {
		return catalogclient.Product{}, f.err
	}
	prod, ok := f.products[id]
	if !ok {
		return catalogclient.Product{}, catalogclient.ErrProductNotFound
	}
	return prod, nil
}

func (f *fakeCatalog) ReserveStock(_ context.Context, _ string, _ []catalogclient.LineItem) error {
	return f.err
}

func (f *fakeCatalog) ReleaseStock(_ context.Context, _ string) error {
	return f.err
}

type fakeCustomer struct {
	customers map[string]customerclient.Customer
	err       error
}

func newFakeCustomer() *fakeCustomer {
	return &fakeCustomer{customers: make(map[string]customerclient.Customer)}
}

func (f *fakeCustomer) EnsureCustomer(_ context.Context, userID string) (customerclient.Customer, error) {
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

func (f *fakeCustomer) GetCustomerByID(_ context.Context, _ string) (customerclient.Customer, error) {
	return customerclient.Customer{}, customerclient.ErrCustomerNotFound
}

func (f *fakeCustomer) GetCustomerByUserID(_ context.Context, userID string) (customerclient.Customer, error) {
	if f.err != nil {
		return customerclient.Customer{}, f.err
	}
	if c, ok := f.customers[userID]; ok {
		return c, nil
	}
	return customerclient.Customer{}, customerclient.ErrCustomerNotFound
}

type fakeOrder struct {
	orders      map[string]orderclient.Order
	err         error
	lastRequest orderclient.PlaceOrderRequest
	prices      map[string]int64
	stock       map[string]int
	placeCalls  int
}

func newFakeOrder() *fakeOrder {
	return &fakeOrder{orders: make(map[string]orderclient.Order)}
}

func (f *fakeOrder) PlaceOrder(_ context.Context, req orderclient.PlaceOrderRequest) (orderclient.Order, error) {
	f.placeCalls++
	f.lastRequest = req
	if f.err != nil {
		return orderclient.Order{}, f.err
	}
	if ord, ok := f.orders[req.IdempotencyKey]; ok {
		return ord, nil
	}
	var total int64
	for _, item := range req.Items {
		if available, tracked := f.stock[item.ProductID]; tracked && available < item.Quantity {
			return orderclient.Order{}, orderclient.ErrInsufficientStock
		}
		total += f.prices[item.ProductID] * int64(item.Quantity)
	}
	ord := orderclient.Order{
		ID:                  "ord-" + req.IdempotencyKey,
		UserID:              req.UserID,
		CustomerID:          "cust-" + req.UserID,
		Status:              "paid",
		TotalCents:          total,
		ShippingQuoteID:     req.ShippingQuoteID,
		ShippingMethodID:    req.ShippingMethodID,
		ShippingMethodName:  req.ShippingMethodName,
		ShippingCostCents:   req.ShippingCostCents,
		DestinationSnapshot: req.DestinationSnapshot,
	}
	f.orders[req.IdempotencyKey] = ord
	return ord, nil
}

func (f *fakeOrder) GetOrder(_ context.Context, id string) (orderclient.Order, error) {
	for _, ord := range f.orders {
		if ord.ID == id {
			return ord, nil
		}
	}
	return orderclient.Order{}, nil
}

type fakeCategory struct {
	categories map[string]categoryclient.CategoryDTO
	err        error
}

func (f *fakeCategory) GetCategory(_ context.Context, id string) (categoryclient.CategoryDTO, error) {
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
	getMethodErr       error
	priceQuoteErr      error
	baseCostCents      int64
	freeThresholdCents int64
}

func (f *fakeShipping) GetQuote(_ context.Context, req shippingclient.GetQuoteRequest) (shippingclient.ShippingQuote, error) {
	return shippingclient.ShippingQuote{
		ID:                  req.ID,
		CustomerID:          "cust-" + strings.TrimPrefix(req.ID, "quote-"),
		MethodID:            "method-1",
		ShippingCostCents:   0,
		DestinationSnapshot: shippingclient.DestinationSnapshot{FullName: "Test User", Street: "1 Main St", City: "Testville", State: "TS", Zip: "12345", Country: "BR"},
		Status:              "active",
	}, nil
}

func (f *fakeShipping) ValidateQuote(_ context.Context, req shippingclient.ValidateQuoteRequest) (shippingclient.ShippingQuote, error) {
	return shippingclient.ShippingQuote{ID: req.ID, CustomerID: req.CustomerID, MethodID: "method-1", Status: "active"}, nil
}

func (f *fakeShipping) PriceQuote(_ context.Context, req shippingclient.PriceQuoteRequest) (shippingclient.ShippingQuote, error) {
	if f.priceQuoteErr != nil {
		return shippingclient.ShippingQuote{}, f.priceQuoteErr
	}
	cost := f.baseCostCents
	if f.freeThresholdCents > 0 && req.SubtotalCents >= f.freeThresholdCents {
		cost = 0
	}
	return shippingclient.ShippingQuote{
		ID:                  req.ID,
		CustomerID:          req.CustomerID,
		MethodID:            "method-1",
		ShippingCostCents:   cost,
		SubtotalCents:       req.SubtotalCents,
		FreeThresholdCents:  f.freeThresholdCents,
		DestinationSnapshot: shippingclient.DestinationSnapshot{FullName: "Test User", Street: "1 Main St", City: "Testville", State: "TS", Zip: "12345", Country: "BR"},
		Status:              "active",
	}, nil
}

func (f *fakeShipping) GetMethod(_ context.Context, id string) (shippingclient.ShippingMethod, error) {
	if f.getMethodErr != nil {
		return shippingclient.ShippingMethod{}, f.getMethodErr
	}
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

// cartEnv wires the cart service over the real MongoDB repository and the
// contract fakes seeded with the features' catalog and shipping fixtures.
type cartEnv struct {
	repo     *cartinfra.CartRepository
	customer *fakeCustomer
	order    *fakeOrder
	shipping *fakeShipping
	modules  *testModules
	svc      *application.Service
}

func newCartEnv(t *testing.T) *cartEnv {
	t.Helper()

	repo := newTestRepo(t)

	catalog := &fakeCatalog{products: map[string]catalogclient.Product{
		featureProductID:      {ID: featureProductID, Name: featureProductName, PriceCents: featureProductPriceCents},
		featureExtraProductID: {ID: featureExtraProductID, Name: featureExtraProductName, PriceCents: featureExtraProductPrice},
	}}
	customer := newFakeCustomer()
	order := newFakeOrder()
	order.prices = map[string]int64{
		featureProductID:      featureProductPriceCents,
		featureExtraProductID: featureExtraProductPrice,
	}
	order.stock = map[string]int{
		featureProductID:      20,
		featureExtraProductID: 20,
	}
	shipping := &fakeShipping{baseCostCents: 599, freeThresholdCents: 5000}
	category := &fakeCategory{categories: map[string]categoryclient.CategoryDTO{
		"cat-1": {ID: "cat-1", Name: "Test Category"},
	}}
	modules := &testModules{
		fakeCatalog:  catalog,
		fakeCustomer: customer,
		fakeOrder:    order,
		fakeCategory: category,
		fakeShipping: shipping,
	}

	return &cartEnv{
		repo:     repo,
		customer: customer,
		order:    order,
		shipping: shipping,
		modules:  modules,
		svc:      application.NewService(repo, modules, metrics.Noop{}),
	}
}

func (e *cartEnv) ensureCustomer(t *testing.T, userID string) customerclient.Customer {
	t.Helper()
	cust, err := e.customer.EnsureCustomer(t.Context(), userID)
	require.NoError(t, err)
	return cust
}

func (e *cartEnv) seedCart(t *testing.T, userID string, version int, items ...domain.Item) domain.Cart {
	t.Helper()
	cust := e.ensureCustomer(t, userID)
	cart, err := domain.NewCart("cart-"+userID, cust.ID)
	require.NoError(t, err)
	cart.Version = version
	cart.Items = items
	require.NoError(t, e.repo.Create(t.Context(), cart))
	return cart
}

func (e *cartEnv) storedCart(t *testing.T, userID string) domain.Cart {
	t.Helper()
	cart, err := e.repo.GetByCustomerIDIncludingConsumed(t.Context(), "cust-"+userID)
	require.NoError(t, err)
	return cart
}

func (e *cartEnv) activeCart(t *testing.T, userID string) domain.Cart {
	t.Helper()
	cart, err := e.repo.GetByCustomerID(t.Context(), "cust-"+userID)
	require.NoError(t, err)
	return cart
}

func (e *cartEnv) consumedCart(t *testing.T, userID string, items ...domain.Item) domain.Cart {
	t.Helper()
	cart := e.seedCart(t, userID, 1, items...)
	require.NoError(t, cart.BeginCheckout(featureCheckoutKey))
	require.NoError(t, cart.MarkConsumed("ord-existing"))
	require.NoError(t, e.repo.Update(t.Context(), cart, 1))
	return cart
}

func (e *cartEnv) pendingCart(t *testing.T, userID, key string, items ...domain.Item) domain.Cart {
	t.Helper()
	cart := e.seedCart(t, userID, 1, items...)
	require.NoError(t, cart.BeginCheckout(key))
	require.NoError(t, e.repo.Update(t.Context(), cart, 1))
	return cart
}

// failingUpdateRepo delegates to the real repository but fails the Nth Update,
// simulating a storage failure after the order has already been placed.
type failingUpdateRepo struct {
	domain.Repository
	failAt int
	calls  int
}

func (r *failingUpdateRepo) Update(ctx context.Context, cart domain.Cart, expectedVersion int) error {
	r.calls++
	if r.calls == r.failAt {
		return errors.New("mongo: cart update failed")
	}
	return r.Repository.Update(ctx, cart, expectedVersion)
}

func featureItem(productID, name string, unitPriceCents int64, quantity int) domain.Item {
	now := time.Now().UTC()
	return domain.Item{
		ProductID:              productID,
		ProductNameSnapshot:    name,
		UnitPriceSnapshotCents: unitPriceCents,
		Quantity:               quantity,
		CreatedAt:              now,
		UpdatedAt:              now,
	}
}

func featureCheckoutRequest(userID, idempotencyKey string) application.CheckoutCartRequest {
	return application.CheckoutCartRequest{
		UserID:          userID,
		IdempotencyKey:  idempotencyKey,
		ShippingQuoteID: "quote-" + userID,
		AddressID:       featureAddressID,
	}
}

func requireItem(t *testing.T, items []domain.Item, productID, name string, quantity int, unitPriceCents int64) {
	t.Helper()
	require.Len(t, items, 1)
	assert.Equal(t, productID, items[0].ProductID)
	assert.Equal(t, name, items[0].ProductNameSnapshot)
	assert.Equal(t, quantity, items[0].Quantity)
	assert.Equal(t, unitPriceCents, items[0].UnitPriceSnapshotCents)
}
