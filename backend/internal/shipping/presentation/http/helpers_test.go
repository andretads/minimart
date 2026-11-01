package http

import (
	"context"
	"encoding/json"
	addressclient "minimart/internal/address/presentation/module/client"
	catalogclient "minimart/internal/catalog/presentation/module/client"
	categoryclient "minimart/internal/category/presentation/module/client"
	customerclient "minimart/internal/customer/presentation/module/client"
	orderclient "minimart/internal/order/presentation/module/client"
	"minimart/internal/shipping/application"
	"minimart/internal/shipping/domain"
	"minimart/internal/shipping/infrastructure/memory"
	shippingclient "minimart/internal/shipping/presentation/module/client"
	"minimart/pkg/auth"
	"minimart/pkg/idempotency"
	"minimart/pkg/metrics"
	"minimart/pkg/outbox"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/require"
)

type stubModules struct {
	catalogclient.CatalogClient
	categoryclient.CategoryClient
	orderclient.OrderClient
	shippingclient.ShippingClient

	addresses map[string]addressclient.Address
	customers map[string]customerclient.Customer
}

func (s *stubModules) GetAddress(_ context.Context, req addressclient.GetAddressRequest) (addressclient.Address, error) {
	addr, ok := s.addresses[req.ID]
	if !ok || addr.CustomerID != req.CustomerID {
		return addressclient.Address{}, addressclient.ErrAddressNotFound
	}
	return addr, nil
}

func (s *stubModules) GetDefaultAddressByCustomerID(ctx context.Context, customerID string) (addressclient.Address, error) {
	return addressclient.Address{}, addressclient.ErrAddressNotFound
}

func (s *stubModules) EnsureCustomer(_ context.Context, userID string) (customerclient.Customer, error) {
	if userID == "" {
		return customerclient.Customer{}, customerclient.ErrCustomerNotFound
	}
	if c, ok := s.customers[userID]; ok {
		return c, nil
	}
	return customerclient.Customer{ID: "cust-" + userID, UserID: userID}, nil
}

func (s *stubModules) GetCustomerByID(_ context.Context, id string) (customerclient.Customer, error) {
	for _, c := range s.customers {
		if c.ID == id {
			return c, nil
		}
	}
	return customerclient.Customer{ID: id, UserID: "user-" + id}, nil
}

func (s *stubModules) GetCustomerByUserID(_ context.Context, userID string) (customerclient.Customer, error) {
	if c, ok := s.customers[userID]; ok {
		return c, nil
	}
	return customerclient.Customer{ID: "cust-" + userID, UserID: userID}, nil
}

type stubOutboxStore struct {
	records []outbox.Record
}

func (s *stubOutboxStore) AddRecordTx(_ context.Context, record outbox.Record, _ pgx.Tx) error {
	s.records = append(s.records, record)
	return nil
}

func (s *stubOutboxStore) GetRecordsByLockID(context.Context, string) ([]outbox.Record, error) {
	return nil, nil
}

func (s *stubOutboxStore) UpdateRecordLockByState(context.Context, string, time.Time, outbox.RecordState) error {
	return nil
}

func (s *stubOutboxStore) UpdateRecordByID(context.Context, outbox.Record) error {
	return nil
}

func (s *stubOutboxStore) ClearLocksWithDurationBeforeDate(context.Context, time.Time) error {
	return nil
}

func (s *stubOutboxStore) ClearLocksByLockID(context.Context, string) error {
	return nil
}

func (s *stubOutboxStore) RemoveRecordsBeforeDatetime(context.Context, time.Time) error {
	return nil
}

type testEnv struct {
	mux     *http.ServeMux
	repo    *memory.Repository
	modules *stubModules
	outbox  *stubOutboxStore
}

func newTestEnv(t *testing.T) *testEnv {
	t.Helper()
	repo := memory.NewRepository()
	modules := &stubModules{
		addresses: map[string]addressclient.Address{},
		customers: map[string]customerclient.Customer{},
	}
	store := &stubOutboxStore{}
	svc := application.NewService(repo, modules, nil, metrics.Noop{}, time.Hour)

	mux := http.NewServeMux()
	NewHandlers(svc, modules).Register(mux)

	return &testEnv{mux: mux, repo: repo, modules: modules, outbox: store}
}

func (e *testEnv) do(t *testing.T, method, path, body string) *httptest.ResponseRecorder {
	t.Helper()
	return e.doAs(t, "", method, path, body)
}

func (e *testEnv) doWithKey(t *testing.T, method, path, body, key string) *httptest.ResponseRecorder {
	t.Helper()
	return e.doAsWithKey(t, "", method, path, body, key)
}

func (e *testEnv) doAs(t *testing.T, userID, method, path, body string) *httptest.ResponseRecorder {
	t.Helper()
	return e.doAsWithKey(t, userID, method, path, body, "")
}

func (e *testEnv) doAsWithKey(t *testing.T, userID, method, path, body, key string) *httptest.ResponseRecorder {
	t.Helper()
	var r *http.Request
	if body == "" {
		r = httptest.NewRequestWithContext(t.Context(), method, path, nil)
	} else {
		r = httptest.NewRequestWithContext(t.Context(), method, path, strings.NewReader(body))
	}
	if userID != "" {
		r = r.WithContext(auth.WithClaims(r.Context(), &auth.Claims{UserID: userID, Subject: userID}))
	}
	if key != "" {
		r.Header.Set(idempotency.Header, key)
	}
	rec := httptest.NewRecorder()
	e.mux.ServeHTTP(rec, r)
	return rec
}

func (e *testEnv) addAddress(customerID string) addressclient.Address {
	addr := addressclient.Address{
		ID:         "addr-1",
		CustomerID: customerID,
		Label:      "Home",
		FullName:   "John Doe",
		Street:     "123 Main St",
		City:       "Sao Paulo",
		State:      "SP",
		Zip:        "01000-000",
		Country:    "BR",
		Phone:      "+5511999999999",
	}
	e.modules.addresses[addr.ID] = addr
	return addr
}

type envelope struct {
	Success bool            `json:"success"`
	Data    json.RawMessage `json:"data"`
	Meta    *struct {
		Page       int `json:"page"`
		PageSize   int `json:"page_size"`
		Total      int `json:"total"`
		TotalPages int `json:"total_pages"`
	} `json:"meta"`
	Error *struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

func decodeEnvelope(t *testing.T, rec *httptest.ResponseRecorder) envelope {
	t.Helper()
	var env envelope
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &env))
	return env
}

func (e *testEnv) seedMethod(t *testing.T, active bool) domain.ShippingMethod {
	t.Helper()
	method, err := domain.NewShippingMethod("Standard", "Standard delivery", active, 3, 5)
	require.NoError(t, err)
	require.NoError(t, e.repo.CreateMethod(t.Context(), method))
	return method
}

func (e *testEnv) seedZone(t *testing.T) domain.ShippingZone {
	t.Helper()
	zone, err := domain.NewShippingZone("Brazil", "BR", "", "", "", "")
	require.NoError(t, err)
	require.NoError(t, e.repo.CreateZone(t.Context(), zone))
	return zone
}

func (e *testEnv) seedRate(t *testing.T, zoneID, methodID string, active bool) domain.ShippingRate {
	t.Helper()
	rate, err := domain.NewShippingRate(zoneID, methodID, 599, 5000, active)
	require.NoError(t, err)
	require.NoError(t, e.repo.CreateRate(t.Context(), rate))
	return rate
}

func (e *testEnv) seedQuote(t *testing.T, customerID string) domain.ShippingQuote {
	t.Helper()
	quote, err := domain.NewShippingQuote(customerID, "method-1", "addr-1", "", domain.DestinationSnapshot{
		FullName: "John Doe",
		Street:   "123 Main St",
		City:     "Sao Paulo",
		Zip:      "01000-000",
		Country:  "BR",
	}, 599, 0, 0, 3, 5)
	require.NoError(t, err)
	require.NoError(t, e.repo.CreateQuote(t.Context(), quote))
	return quote
}

func (e *testEnv) seedShipment(t *testing.T, userID, orderID string, status domain.ShipmentStatus) domain.Shipment {
	t.Helper()
	shipment, err := domain.NewShipment(orderID, userID, "method-1", domain.DestinationSnapshot{
		FullName: "John Doe",
		Street:   "123 Main St",
		City:     "Sao Paulo",
		Zip:      "01000-000",
		Country:  "BR",
	}, 599)
	require.NoError(t, err)

	switch status {
	case domain.ShipmentStatusProcessing:
		require.NoError(t, shipment.Process())
	case domain.ShipmentStatusShipped:
		require.NoError(t, shipment.Process())
		require.NoError(t, shipment.Dispatch("TRACK-1", "Correios"))
	case domain.ShipmentStatusDelivered:
		require.NoError(t, shipment.Process())
		require.NoError(t, shipment.Dispatch("TRACK-1", "Correios"))
		require.NoError(t, shipment.Deliver())
	}

	require.NoError(t, e.repo.CreateShipment(t.Context(), shipment))
	return shipment
}

func shipmentIDs(shipments []ShipmentResponse) []string {
	ids := make([]string, len(shipments))
	for i, s := range shipments {
		ids[i] = s.ID
	}
	return ids
}
