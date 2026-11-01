package application

import (
	"context"
	"minimart/internal/cart/domain"
	"minimart/pkg/metrics"
	"strings"

	addressclient "minimart/internal/address/presentation/module/client"
	catalogclient "minimart/internal/catalog/presentation/module/client"
	categoryclient "minimart/internal/category/presentation/module/client"
	customerclient "minimart/internal/customer/presentation/module/client"
	orderclient "minimart/internal/order/presentation/module/client"
	shippingclient "minimart/internal/shipping/presentation/module/client"
)

type fakeRepo struct {
	carts map[string]domain.Cart
}

func newFakeRepo() *fakeRepo {
	return &fakeRepo{carts: make(map[string]domain.Cart)}
}

func (f *fakeRepo) GetByCustomerID(ctx context.Context, customerID string) (domain.Cart, error) {
	for _, cart := range f.carts {
		if cart.CustomerID == customerID && (cart.Status == domain.StatusActive || cart.Status == domain.StatusCheckoutPending) {
			return cart, nil
		}
	}
	return domain.Cart{}, domain.ErrCartNotFound
}

func (f *fakeRepo) GetByCustomerIDIncludingConsumed(ctx context.Context, customerID string) (domain.Cart, error) {
	for _, cart := range f.carts {
		if cart.CustomerID == customerID {
			return cart, nil
		}
	}
	return domain.Cart{}, domain.ErrCartNotFound
}

func (f *fakeRepo) Create(ctx context.Context, cart domain.Cart) error {
	key := cart.ID
	if _, exists := f.carts[key]; exists {
		return domain.ErrCartAlreadyExists
	}
	f.carts[key] = cart
	return nil
}

func (f *fakeRepo) Update(ctx context.Context, cart domain.Cart, expectedVersion int) error {
	existing, ok := f.carts[cart.ID]
	if !ok {
		return domain.ErrCartNotFound
	}
	if existing.Version != expectedVersion {
		return domain.ErrCartOptimisticLock
	}
	cart.Version = expectedVersion + 1
	f.carts[cart.ID] = cart
	return nil
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
}

func newFakeOrder() *fakeOrder {
	return &fakeOrder{orders: make(map[string]orderclient.Order)}
}

func (f *fakeOrder) PlaceOrder(ctx context.Context, req orderclient.PlaceOrderRequest) (orderclient.Order, error) {
	f.lastRequest = req
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

func newTestService() (*fakeRepo, *fakeCatalog, *fakeCustomer, *fakeOrder, *fakeCategory, *Service) {
	repo := newFakeRepo()
	catalog := newFakeCatalog()
	catalog.products["prod-1"] = catalogclient.Product{ID: "prod-1", Name: "Product 1", PriceCents: 1000}
	customer := newFakeCustomer()
	order := newFakeOrder()
	category := newFakeCategory()
	category.categories["cat-1"] = categoryclient.CategoryDTO{ID: "cat-1", Name: "Test Category"}
	mod := &fakeModules{fakeCatalog: catalog, fakeCustomer: customer, fakeOrder: order, fakeCategory: category, fakeShipping: &fakeShipping{}}
	svc := NewService(repo, mod, metrics.Noop{})
	return repo, catalog, customer, order, category, svc
}
