package http

import (
	"context"
	"encoding/json"
	"minimart/internal/catalog/application"
	"minimart/internal/catalog/infrastructure/memory"
	"minimart/internal/catalog/testkit"
	"minimart/internal/category/presentation/module/client"
	customercontract "minimart/internal/customer/presentation/module/client"
	"minimart/pkg/auth"
	"minimart/pkg/metrics"
	"minimart/pkg/moderation"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// fakeCategoryProvider is a test double for CategoryProvider.
type fakeCategoryProvider struct {
	err error
}

func (f fakeCategoryProvider) GetCategory(ctx context.Context, id string) (client.CategoryDTO, error) {
	if f.err != nil {
		return client.CategoryDTO{}, f.err
	}
	return client.CategoryDTO{ID: id, Name: "Test Category"}, nil
}

type envelope struct {
	Success bool            `json:"success"`
	Data    json.RawMessage `json:"data"`
	Error   *struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

// fakeCustomerClient stands in for the customer contract. It records the user
// id it was asked to resolve so tests can assert the session was used.
type fakeCustomerClient struct {
	customer customercontract.Customer
	err      error
	gotUser  string
}

func (f *fakeCustomerClient) EnsureCustomer(_ context.Context, userID string) (customercontract.Customer, error) {
	f.gotUser = userID
	if f.err != nil {
		return customercontract.Customer{}, f.err
	}
	return f.customer, nil
}

func (f *fakeCustomerClient) GetCustomerByID(_ context.Context, _ string) (customercontract.Customer, error) {
	return f.customer, f.err
}

func (f *fakeCustomerClient) GetCustomerByUserID(_ context.Context, _ string) (customercontract.Customer, error) {
	return f.customer, f.err
}

func newTestMux(t *testing.T) (*http.ServeMux, *memory.Repository) {
	t.Helper()
	mux, repo, _ := newTestMuxWithCustomer(t)
	return mux, repo
}

func newTestMuxWithCustomer(t *testing.T) (*http.ServeMux, *memory.Repository, *fakeCustomerClient) {
	t.Helper()
	repo := memory.New()
	svc := application.NewService(repo, testkit.NewFakeS3Client(), fakeCategoryProvider{}, time.Hour, 5*1024*1024, metrics.Noop{}, moderation.NewNoopProvider())
	customer := &fakeCustomerClient{customer: customercontract.Customer{ID: "cust-1", UserID: "user-1"}}
	mux := http.NewServeMux()
	NewHandlers(svc, customer).Register(mux)
	return mux, repo, customer
}

func do(t *testing.T, mux *http.ServeMux, method, path, body string) *httptest.ResponseRecorder {
	t.Helper()
	var r *http.Request
	if body == "" {
		r = httptest.NewRequestWithContext(t.Context(), method, path, nil)
	} else {
		r = httptest.NewRequestWithContext(t.Context(), method, path, strings.NewReader(body))
	}
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, r)
	return rec
}

func doWithHeaders(t *testing.T, mux *http.ServeMux, method, path, body string, headers map[string]string) *httptest.ResponseRecorder {
	t.Helper()
	var r *http.Request
	if body == "" {
		r = httptest.NewRequestWithContext(t.Context(), method, path, nil)
	} else {
		r = httptest.NewRequestWithContext(t.Context(), method, path, strings.NewReader(body))
	}
	for k, v := range headers {
		r.Header.Set(k, v)
	}
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, r)
	return rec
}

// doAuthenticated issues a request carrying the auth claims of an
// authenticated session.
func doAuthenticated(t *testing.T, mux *http.ServeMux, method, path, body, userID string) *httptest.ResponseRecorder {
	t.Helper()
	var r *http.Request
	if body == "" {
		r = httptest.NewRequestWithContext(t.Context(), method, path, nil)
	} else {
		r = httptest.NewRequestWithContext(t.Context(), method, path, strings.NewReader(body))
	}
	ctx := auth.WithClaims(r.Context(), &auth.Claims{UserID: userID})
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, r.WithContext(ctx))
	return rec
}
