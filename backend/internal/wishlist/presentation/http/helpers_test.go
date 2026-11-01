package wishlisthttp

import (
	"bytes"
	"context"
	"encoding/json"
	"minimart/internal/wishlist/application"
	"minimart/internal/wishlist/domain"
	"minimart/pkg/auth"
	"minimart/pkg/metrics"
	"net/http"
	"net/http/httptest"
	"testing"

	addressclient "minimart/internal/address/presentation/module/client"
	catalogclient "minimart/internal/catalog/presentation/module/client"
	categoryclient "minimart/internal/category/presentation/module/client"
	customerclient "minimart/internal/customer/presentation/module/client"
	orderclient "minimart/internal/order/presentation/module/client"
	shippingclient "minimart/internal/shipping/presentation/module/client"

	"github.com/stretchr/testify/require"
)

type testModules struct {
	*fakeCatalog
	*fakeCustomer
	*fakeOrder
	*fakeCategory
	shippingclient.ShippingClient
	addressclient.AddressClient
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
	orders map[string]orderclient.Order
	err    error
}

func newFakeOrder() *fakeOrder {
	return &fakeOrder{orders: make(map[string]orderclient.Order)}
}

func (f *fakeOrder) PlaceOrder(_ context.Context, req orderclient.PlaceOrderRequest) (orderclient.Order, error) {
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

type fakeRepo struct {
	wishlists map[string]domain.Wishlist
}

func newFakeRepo() *fakeRepo {
	return &fakeRepo{wishlists: make(map[string]domain.Wishlist)}
}

func (f *fakeRepo) GetByCustomerID(_ context.Context, customerID string) (domain.Wishlist, error) {
	for _, wishlist := range f.wishlists {
		if wishlist.CustomerID == customerID {
			return wishlist, nil
		}
	}
	return domain.Wishlist{}, domain.ErrWishlistNotFound
}

func (f *fakeRepo) Create(_ context.Context, wishlist domain.Wishlist) error {
	key := wishlist.ID
	if _, exists := f.wishlists[key]; exists {
		return domain.ErrWishlistAlreadyExists
	}
	f.wishlists[key] = wishlist
	return nil
}

func (f *fakeRepo) Update(_ context.Context, wishlist domain.Wishlist) error {
	existing, ok := f.wishlists[wishlist.ID]
	if !ok {
		return domain.ErrWishlistNotFound
	}
	if existing.CustomerID != wishlist.CustomerID {
		return domain.ErrWishlistNotFound
	}
	f.wishlists[wishlist.ID] = wishlist
	return nil
}

func newTestMux(t *testing.T) *http.ServeMux {
	t.Helper()
	repo := newFakeRepo()
	catalog := &fakeCatalog{products: map[string]catalogclient.Product{
		"prod-1": {ID: "prod-1", Name: "Product 1", PriceCents: 1000},
		"prod-2": {ID: "prod-2", Name: "Product 2", PriceCents: 2000},
	}}
	customer := newFakeCustomer()
	order := newFakeOrder()
	category := &fakeCategory{categories: map[string]categoryclient.CategoryDTO{
		"cat-1": {ID: "cat-1", Name: "Test Category"},
	}}
	mod := &testModules{fakeCatalog: catalog, fakeCustomer: customer, fakeOrder: order, fakeCategory: category}
	svc := application.NewService(repo, mod, metrics.Noop{})
	h := NewHandlers(svc)
	mux := http.NewServeMux()
	h.Register(mux)
	return mux
}

func doRequest(t *testing.T, mux *http.ServeMux, method, path string, body []byte, headers map[string]string) *httptest.ResponseRecorder {
	t.Helper()
	req, err := http.NewRequestWithContext(t.Context(), method, path, bytes.NewBuffer(body))
	require.NoError(t, err)
	req = req.WithContext(auth.WithClaims(req.Context(), &auth.Claims{UserID: "user-1"}))
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	return rec
}

func doRequestUnauth(t *testing.T, mux *http.ServeMux, method, path string, body []byte, headers map[string]string) *httptest.ResponseRecorder {
	t.Helper()
	req, err := http.NewRequestWithContext(t.Context(), method, path, bytes.NewBuffer(body))
	require.NoError(t, err)
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	return rec
}

func decodeResponse(t *testing.T, rec *httptest.ResponseRecorder, target any) {
	t.Helper()
	require.NoError(t, json.NewDecoder(rec.Body).Decode(target))
}
