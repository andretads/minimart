package http

import (
	"context"
	"encoding/json"
	"errors"
	"minimart/internal/order/application"
	"minimart/internal/order/infrastructure/memory"
	orderclient "minimart/internal/order/presentation/module/client"
	"minimart/pkg/idempotency"
	"minimart/pkg/metrics"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	addressclient "minimart/internal/address/presentation/module/client"
	catalogclient "minimart/internal/catalog/presentation/module/client"
	categoryclient "minimart/internal/category/presentation/module/client"
	customerclient "minimart/internal/customer/presentation/module/client"
	shippingclient "minimart/internal/shipping/presentation/module/client"
)

type stubModules struct {
	products map[string]catalogclient.Product
	shippingclient.ShippingClient
	addressclient.AddressClient
}

func (s stubModules) GetProduct(_ context.Context, id string) (catalogclient.Product, error) {
	p, ok := s.products[id]
	if !ok {
		return catalogclient.Product{}, catalogclient.ErrProductNotFound
	}
	return p, nil
}

func (s stubModules) GetCategory(_ context.Context, id string) (categoryclient.CategoryDTO, error) {
	return categoryclient.CategoryDTO{ID: id, Name: "Test Category"}, nil
}

func (s stubModules) ReserveStock(context.Context, string, []catalogclient.LineItem) error {
	return nil
}

func (s stubModules) ReleaseStock(context.Context, string) error { return nil }

func (s stubModules) EnsureCustomer(_ context.Context, userID string) (customerclient.Customer, error) {
	if userID == "" {
		return customerclient.Customer{}, errors.New("customer: user_id é obrigatório")
	}
	return customerclient.Customer{ID: "cust-" + userID, UserID: userID}, nil
}

func (s stubModules) GetCustomerByID(_ context.Context, id string) (customerclient.Customer, error) {
	return customerclient.Customer{ID: id, UserID: "user-" + id}, nil
}

func (s stubModules) GetCustomerByUserID(_ context.Context, userID string) (customerclient.Customer, error) {
	return customerclient.Customer{ID: "cust-" + userID, UserID: userID}, nil
}

func (s stubModules) PlaceOrder(_ context.Context, req orderclient.PlaceOrderRequest) (orderclient.Order, error) {
	return orderclient.Order{}, nil
}

func (s stubModules) GetOrder(_ context.Context, id string) (orderclient.Order, error) {
	return orderclient.Order{}, nil
}

type stubIdempotency struct {
	records map[string]idempotency.Record
}

func newStubIdempotency() *stubIdempotency {
	return &stubIdempotency{records: map[string]idempotency.Record{}}
}

func (s *stubIdempotency) Get(_ context.Context, key string) (idempotency.Record, bool, error) {
	r, ok := s.records[key]
	return r, ok, nil
}

func (s *stubIdempotency) Reserve(_ context.Context, key, fp string, _ time.Duration) (bool, error) {
	if _, ok := s.records[key]; ok {
		return false, nil
	}
	s.records[key] = idempotency.Record{Fingerprint: fp}
	return true, nil
}

func (s *stubIdempotency) Complete(_ context.Context, key, fp, resourceID string, _ time.Duration) error {
	s.records[key] = idempotency.Record{Fingerprint: fp, ResourceID: resourceID}
	return nil
}

func (s *stubIdempotency) Release(_ context.Context, key, _ string) error {
	delete(s.records, key)
	return nil
}

type envelope struct {
	Success bool            `json:"success"`
	Data    json.RawMessage `json:"data"`
	Error   *struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

func newTestMux(t *testing.T) *http.ServeMux {
	t.Helper()
	modules := stubModules{products: map[string]catalogclient.Product{
		"p1": {ID: "p1", Name: "Widget", PriceCents: 500},
		"p2": {ID: "p2", Name: "Gadget", PriceCents: 250},
	}}
	svc := application.NewService(memory.New(), modules, newStubIdempotency(), metrics.Noop{}, time.Hour)
	mux := http.NewServeMux()
	NewHandlers(svc).Register(mux)
	return mux
}

func do(t *testing.T, mux *http.ServeMux, method, path, body string) *httptest.ResponseRecorder {
	t.Helper()
	return doWithKey(t, mux, method, path, body, "test-idempotency-key")
}

func doWithKey(t *testing.T, mux *http.ServeMux, method, path, body, key string) *httptest.ResponseRecorder {
	t.Helper()
	var r *http.Request
	if body == "" {
		r = httptest.NewRequestWithContext(t.Context(), method, path, nil)
	} else {
		r = httptest.NewRequestWithContext(t.Context(), method, path, strings.NewReader(body))
	}
	if key != "" {
		r.Header.Set(idempotency.Header, key)
	}
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, r)
	return rec
}
