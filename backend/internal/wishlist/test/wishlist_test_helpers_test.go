//go:build integration

package test

import (
	"context"
	"minimart/internal/wishlist/application"
	"minimart/internal/wishlist/domain"
	"minimart/pkg/metrics"
	"testing"

	wishlistinfra "minimart/internal/wishlist/infrastructure/mongo"

	addressclient "minimart/internal/address/presentation/module/client"
	catalogclient "minimart/internal/catalog/presentation/module/client"
	categoryclient "minimart/internal/category/presentation/module/client"
	customerclient "minimart/internal/customer/presentation/module/client"
	orderclient "minimart/internal/order/presentation/module/client"
	shippingclient "minimart/internal/shipping/presentation/module/client"

	mongodriver "go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	"github.com/stretchr/testify/require"
)

func newTestRepo(t *testing.T) *wishlistinfra.Repository {
	t.Helper()
	client, err := mongodriver.Connect(options.Client().ApplyURI(testMongoURI))
	require.NoError(t, err)
	t.Cleanup(func() { _ = client.Disconnect(t.Context()) })

	repo := wishlistinfra.NewRepository(client.Database("minimart"))
	require.NoError(t, repo.EnsureIndexes(t.Context()))
	return repo
}

func newWishlistService(t *testing.T) *application.Service {
	t.Helper()
	repo := newTestRepo(t)

	catalog := &fakeCatalog{products: map[string]catalogclient.Product{
		"prod-1": {ID: "prod-1", Name: "Product 1", PriceCents: 1000},
		"prod-2": {ID: "prod-2", Name: "Product 2", PriceCents: 2000},
		"prod-3": {ID: "prod-3", Name: "Product 3", PriceCents: 3000},
	}}
	customer := newFakeCustomer()
	order := newFakeOrder()
	category := newFakeCategory()
	category.categories["cat-1"] = categoryclient.CategoryDTO{ID: "cat-1", Name: "Test Category"}

	mod := &testModules{fakeCatalog: catalog, fakeCustomer: customer, fakeOrder: order, fakeCategory: category}
	return application.NewService(repo, mod, metrics.Noop{})
}

// contract fakes
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

var (
	_ = domain.NewWishlist
	_ = mongodriver.Connect
	_ = options.Client
)
