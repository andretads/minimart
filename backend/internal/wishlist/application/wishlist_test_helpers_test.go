package application

import (
	"context"
	"minimart/internal/wishlist/infrastructure/memory"
	"minimart/pkg/metrics"
	"sync"
	"testing"

	addressclient "minimart/internal/address/presentation/module/client"
	catalogclient "minimart/internal/catalog/presentation/module/client"
	categoryclient "minimart/internal/category/presentation/module/client"
	customerclient "minimart/internal/customer/presentation/module/client"
	orderclient "minimart/internal/order/presentation/module/client"
	shippingclient "minimart/internal/shipping/presentation/module/client"
)

type fakeCatalog struct {
	mu       sync.RWMutex
	products map[string]catalogclient.Product
	err      error
}

func newFakeCatalog() *fakeCatalog {
	return &fakeCatalog{products: make(map[string]catalogclient.Product)}
}

func (f *fakeCatalog) GetProduct(ctx context.Context, id string) (catalogclient.Product, error) {
	f.mu.RLock()
	defer f.mu.RUnlock()
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
	f.mu.RLock()
	defer f.mu.RUnlock()
	return f.err
}

func (f *fakeCatalog) ReleaseStock(ctx context.Context, orderID string) error {
	f.mu.RLock()
	defer f.mu.RUnlock()
	return f.err
}

type fakeCustomer struct {
	mu        sync.RWMutex
	customers map[string]customerclient.Customer
	err       error
}

func newFakeCustomer() *fakeCustomer {
	return &fakeCustomer{customers: make(map[string]customerclient.Customer)}
}

func (f *fakeCustomer) EnsureCustomer(ctx context.Context, userID string) (customerclient.Customer, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
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
	f.mu.RLock()
	defer f.mu.RUnlock()
	return customerclient.Customer{}, customerclient.ErrCustomerNotFound
}

func (f *fakeCustomer) GetCustomerByUserID(ctx context.Context, userID string) (customerclient.Customer, error) {
	f.mu.RLock()
	defer f.mu.RUnlock()
	if f.err != nil {
		return customerclient.Customer{}, f.err
	}
	if c, ok := f.customers[userID]; ok {
		return c, nil
	}
	return customerclient.Customer{}, customerclient.ErrCustomerNotFound
}

type fakeOrder struct {
	mu     sync.RWMutex
	orders map[string]orderclient.Order
	err    error
}

func newFakeOrder() *fakeOrder {
	return &fakeOrder{orders: make(map[string]orderclient.Order)}
}

func (f *fakeOrder) PlaceOrder(ctx context.Context, req orderclient.PlaceOrderRequest) (orderclient.Order, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
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
	f.mu.RLock()
	defer f.mu.RUnlock()
	if f.err != nil {
		return orderclient.Order{}, f.err
	}
	for _, ord := range f.orders {
		if ord.ID == id {
			return ord, nil
		}
	}
	return orderclient.Order{}, nil
}

type fakeCategory struct {
	mu         sync.RWMutex
	categories map[string]categoryclient.CategoryDTO
	err        error
}

func newFakeCategory() *fakeCategory {
	return &fakeCategory{categories: make(map[string]categoryclient.CategoryDTO)}
}

func (f *fakeCategory) GetCategory(ctx context.Context, id string) (categoryclient.CategoryDTO, error) {
	f.mu.RLock()
	defer f.mu.RUnlock()
	if f.err != nil {
		return categoryclient.CategoryDTO{}, f.err
	}
	cat, ok := f.categories[id]
	if !ok {
		return categoryclient.CategoryDTO{}, categoryclient.ErrCategoryNotFound
	}
	return cat, nil
}

type fakeModules struct {
	*fakeCatalog
	*fakeCustomer
	*fakeOrder
	*fakeCategory
	shippingclient.ShippingClient
	addressclient.AddressClient
}

func newTestService(t *testing.T) (*Service, *memory.Repository, *fakeCatalog, *fakeCustomer) {
	t.Helper()
	repo := memory.New()
	catalog := newFakeCatalog()
	for _, id := range []string{"prod-1", "prod-2", "prod-3"} {
		catalog.products[id] = catalogclient.Product{ID: id, Name: "Product " + id, PriceCents: 1000}
	}
	customer := newFakeCustomer()
	order := newFakeOrder()
	category := newFakeCategory()
	category.categories["cat-1"] = categoryclient.CategoryDTO{ID: "cat-1", Name: "Test Category"}
	mod := &fakeModules{fakeCatalog: catalog, fakeCustomer: customer, fakeOrder: order, fakeCategory: category}
	svc := NewService(repo, mod, metrics.Noop{})
	return svc, repo, catalog, customer
}
