package http

import (
	"context"
	"encoding/json"
	"minimart/internal/address/application"
	"minimart/internal/address/infrastructure/memory"
	"minimart/internal/customer/presentation/module/client"
	"minimart/pkg/auth"
	"minimart/pkg/metrics"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

type envelope struct {
	Success bool            `json:"success"`
	Data    json.RawMessage `json:"data"`
	Error   *struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

// stubCustomerClient is a test stub for the CustomerClient interface.
type stubCustomerClient struct {
	customerID string
	err        error
}

func (s *stubCustomerClient) EnsureCustomer(ctx context.Context, userID string) (client.Customer, error) {
	if s.err != nil {
		return client.Customer{}, s.err
	}
	if s.customerID == "" {
		s.customerID = "cust-" + userID
	}
	return client.Customer{ID: s.customerID, UserID: userID}, nil
}

func (s *stubCustomerClient) GetCustomerByID(ctx context.Context, id string) (client.Customer, error) {
	return client.Customer{}, nil
}

func (s *stubCustomerClient) GetCustomerByUserID(ctx context.Context, userID string) (client.Customer, error) {
	return client.Customer{}, nil
}

func newTestMux(t *testing.T) (*http.ServeMux, *memory.Repository) {
	t.Helper()
	repo := memory.New()
	svc := application.NewService(repo, metrics.Noop{})
	customerClient := &stubCustomerClient{}
	mux := http.NewServeMux()
	NewHandlers(svc, customerClient).Register(mux)
	return mux, repo
}

func doWithContextAndHeaders(t *testing.T, mux *http.ServeMux, method, path, body string, ctx context.Context, headers map[string]string) *httptest.ResponseRecorder {
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
	if ctx != nil {
		r = r.WithContext(ctx)
	}
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, r)
	return rec
}

func withAuthContext(ctx context.Context, userID string) context.Context {
	t := &testing.T{}
	t.Helper()
	claims := &auth.Claims{
		UserID:  userID,
		Subject: userID,
	}
	return auth.WithClaims(ctx, claims)
}

func assertJSONResponse(t *testing.T, rec *httptest.ResponseRecorder, expectedStatus int, expectedSuccess bool) envelope {
	t.Helper()
	require.Equal(t, expectedStatus, rec.Code)
	var env envelope
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &env))
	require.Equal(t, expectedSuccess, env.Success)
	return env
}

func buildCreateAddressRequest(t *testing.T, label, fullName, street, city, state, zip, country, phone string, isDefault bool) string {
	t.Helper()
	req := createAddressRequest{
		Label:     label,
		FullName:  fullName,
		Street:    street,
		City:      city,
		State:     state,
		Zip:       zip,
		Country:   country,
		Phone:     phone,
		IsDefault: isDefault,
	}
	b, err := json.Marshal(req)
	require.NoError(t, err)
	return string(b)
}

func buildUpdateAddressRequest(t *testing.T, label, fullName, street, city, state, zip, country, phone string, version int) string {
	t.Helper()
	req := updateAddressRequest{
		Label:    label,
		FullName: fullName,
		Street:   street,
		City:     city,
		State:    state,
		Zip:      zip,
		Country:  country,
		Phone:    phone,
		Version:  version,
	}
	b, err := json.Marshal(req)
	require.NoError(t, err)
	return string(b)
}
