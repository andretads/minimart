package application

import (
	"context"
	"errors"
	"minimart/internal/cart/domain"
	"minimart/internal/cart/infrastructure/memory"
	"minimart/pkg/metrics"
	"strings"
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

// The logical fixtures from features/cart/*.feature. The feature calls the
// customer "u1"/"u2"; the contract layer derives the customer id from the user
// id ("cust-"+user), and the shipping quote fixture is named after the user.
const (
	featureUser1      = "user-1"
	featureUser2      = "user-2"
	featureUserRace   = "user-race"
	featureUserNoCart = "user-no-cart"

	featureProductID         = "p1"
	featureProductName       = "Widget"
	featureProductPriceCents = int64(500)

	featureExtraProductID    = "p2"
	featureExtraProductName  = "Gadget"
	featureExtraProductPrice = int64(700)

	featureAddressID   = "addr-1"
	featureCheckoutKey = "k1"
)

// failingUpdateRepo wraps the in-memory repository and fails the Nth Update,
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

type fakeCatalog struct {
	products map[string]catalogclient.Product
	err      error
}

func newFakeCatalog() *fakeCatalog {
	return &fakeCatalog{products: make(map[string]catalogclient.Product)}
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

func (f *fakeOrder) PlaceOrder(ctx context.Context, req orderclient.PlaceOrderRequest) (orderclient.Order, error) {
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
	return shippingclient.ShippingQuote{
		ID:         req.ID,
		CustomerID: req.CustomerID,
		MethodID:   "method-1",
		Status:     "active",
	}, nil
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

type fakeModules struct {
	*fakeCatalog
	*fakeCustomer
	*fakeOrder
	*fakeCategory
	*fakeShipping
	addressclient.AddressClient
}

// featureEnv wires the service over the in-memory repository and the contract
// fakes seeded with the features' catalog and shipping fixtures.
type featureEnv struct {
	repo     *memory.Repository
	customer *fakeCustomer
	order    *fakeOrder
	shipping *fakeShipping
	modules  *fakeModules
	svc      *Service
}

func newFeatureEnv(t *testing.T) *featureEnv {
	t.Helper()

	repo := memory.New()

	catalog := newFakeCatalog()
	catalog.products[featureProductID] = catalogclient.Product{
		ID:         featureProductID,
		Name:       featureProductName,
		PriceCents: featureProductPriceCents,
	}
	catalog.products[featureExtraProductID] = catalogclient.Product{
		ID:         featureExtraProductID,
		Name:       featureExtraProductName,
		PriceCents: featureExtraProductPrice,
	}

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

	category := newFakeCategory()
	category.categories["cat-1"] = categoryclient.CategoryDTO{ID: "cat-1", Name: "Test Category"}

	mod := &fakeModules{
		fakeCatalog:  catalog,
		fakeCustomer: customer,
		fakeOrder:    order,
		fakeCategory: category,
		fakeShipping: shipping,
	}

	return &featureEnv{
		repo:     repo,
		customer: customer,
		order:    order,
		shipping: shipping,
		modules:  mod,
		svc:      NewService(repo, mod, metrics.Noop{}),
	}
}

// ensureCustomer provisions the customer the feature's quotes belong to.
func (e *featureEnv) ensureCustomer(t *testing.T, userID string) customerclient.Customer {
	t.Helper()
	cust, err := e.customer.EnsureCustomer(t.Context(), userID)
	require.NoError(t, err)
	return cust
}

// seedCart persists a cart with the given items and version straight through
// the repository so a feature can describe the storage it starts from.
func (e *featureEnv) seedCart(t *testing.T, userID string, version int, items ...domain.Item) domain.Cart {
	t.Helper()
	cust := e.ensureCustomer(t, userID)
	cart, err := domain.NewCart("cart-"+userID, cust.ID)
	require.NoError(t, err)
	cart.Version = version
	cart.Items = items
	require.NoError(t, e.repo.Create(t.Context(), cart))
	return cart
}

func (e *featureEnv) storedCart(t *testing.T, userID string) domain.Cart {
	t.Helper()
	cart, err := e.repo.GetByCustomerIDIncludingConsumed(t.Context(), "cust-"+userID)
	require.NoError(t, err)
	return cart
}

func (e *featureEnv) activeCart(t *testing.T, userID string) domain.Cart {
	t.Helper()
	cart, err := e.repo.GetByCustomerID(t.Context(), "cust-"+userID)
	require.NoError(t, err)
	return cart
}

func (e *featureEnv) consumedCart(t *testing.T, userID string, items ...domain.Item) domain.Cart {
	t.Helper()
	cart := e.seedCart(t, userID, 1, items...)
	require.NoError(t, cart.BeginCheckout(featureCheckoutKey))
	require.NoError(t, cart.MarkConsumed("ord-existing"))
	require.NoError(t, e.repo.Update(t.Context(), cart, 1))
	return cart
}

func (e *featureEnv) pendingCart(t *testing.T, userID, key string, items ...domain.Item) domain.Cart {
	t.Helper()
	cart := e.seedCart(t, userID, 1, items...)
	require.NoError(t, cart.BeginCheckout(key))
	require.NoError(t, e.repo.Update(t.Context(), cart, 1))
	return cart
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

func featureCheckoutRequest(userID, idempotencyKey string) CheckoutCartRequest {
	return CheckoutCartRequest{
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
