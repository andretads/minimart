package application

import (
	"context"
	"errors"
	addressclient "minimart/internal/address/presentation/module/client"
	catalogclient "minimart/internal/catalog/presentation/module/client"
	categoryclient "minimart/internal/category/presentation/module/client"
	customerclient "minimart/internal/customer/presentation/module/client"
	orderclient "minimart/internal/order/presentation/module/client"
	"minimart/internal/shipping/domain"
	"minimart/internal/shipping/infrastructure/memory"
	shippingclient "minimart/internal/shipping/presentation/module/client"
	"minimart/pkg/idempotency"
	"minimart/pkg/metrics"
	"minimart/pkg/module/contracts"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

type FakeModules struct {
	Addresses  map[string]addressclient.Address
	Customers  map[string]customerclient.Customer
	Products   map[string]catalogclient.Product
	Categories map[string]categoryclient.CategoryDTO
	Orders     map[string]orderclient.Order
	Methods    map[string]shippingclient.ShippingMethod
	Zones      map[string]shippingclient.ShippingZone
	Rates      map[string]shippingclient.ShippingRate
}

func (f *FakeModules) GetAddress(_ context.Context, req addressclient.GetAddressRequest) (addressclient.Address, error) {
	addr, ok := f.Addresses[req.ID]
	if !ok || (req.CustomerID != "" && addr.CustomerID != req.CustomerID) {
		return addressclient.Address{}, addressclient.ErrAddressNotFound
	}
	return addr, nil
}

func (f *FakeModules) GetDefaultAddressByCustomerID(context.Context, string) (addressclient.Address, error) {
	return addressclient.Address{}, addressclient.ErrAddressNotFound
}

func (f *FakeModules) GetProduct(_ context.Context, id string) (catalogclient.Product, error) {
	if p, ok := f.Products[id]; ok {
		return p, nil
	}
	return catalogclient.Product{}, catalogclient.ErrProductNotFound
}

func (f *FakeModules) ReserveStock(context.Context, string, []catalogclient.LineItem) error {
	return nil
}

func (f *FakeModules) ReleaseStock(context.Context, string) error { return nil }

func (f *FakeModules) EnsureCustomer(_ context.Context, userID string) (customerclient.Customer, error) {
	if userID == "" {
		return customerclient.Customer{}, errors.New("customer: user_id é obrigatório")
	}
	if c, ok := f.Customers[userID]; ok {
		return c, nil
	}
	return customerclient.Customer{ID: "cust-" + userID, UserID: userID}, nil
}

func (f *FakeModules) GetCustomerByID(_ context.Context, id string) (customerclient.Customer, error) {
	if c, ok := f.Customers[id]; ok {
		return c, nil
	}
	return customerclient.Customer{ID: id, UserID: "user-" + id}, nil
}

func (f *FakeModules) GetCustomerByUserID(_ context.Context, userID string) (customerclient.Customer, error) {
	if c, ok := f.Customers[userID]; ok {
		return c, nil
	}
	return customerclient.Customer{ID: "cust-" + userID, UserID: userID}, nil
}

func (f *FakeModules) GetCategory(_ context.Context, id string) (categoryclient.CategoryDTO, error) {
	if cat, ok := f.Categories[id]; ok {
		return cat, nil
	}
	return categoryclient.CategoryDTO{ID: id, Name: "Test Category"}, nil
}

func (f *FakeModules) GetOrder(_ context.Context, id string) (orderclient.Order, error) {
	if o, ok := f.Orders[id]; ok {
		return o, nil
	}
	return orderclient.Order{}, errors.New("pedido não encontrado")
}

func (f *FakeModules) PlaceOrder(context.Context, orderclient.PlaceOrderRequest) (orderclient.Order, error) {
	return orderclient.Order{}, nil
}

func (f *FakeModules) ListMethods(context.Context) ([]shippingclient.ShippingMethod, error) {
	methods := make([]shippingclient.ShippingMethod, 0, len(f.Methods))
	for _, m := range f.Methods {
		methods = append(methods, m)
	}
	return methods, nil
}

func (f *FakeModules) GetMethod(_ context.Context, id string) (shippingclient.ShippingMethod, error) {
	if m, ok := f.Methods[id]; ok {
		return m, nil
	}
	return shippingclient.ShippingMethod{}, shippingclient.ErrMethodNotFound
}

func (f *FakeModules) CreateMethod(_ context.Context, req shippingclient.CreateMethodRequest) (shippingclient.ShippingMethod, error) {
	method := shippingclient.ShippingMethod{ID: "test-" + req.Name, Name: req.Name, Description: req.Description, Active: req.Active, EstimatedMinDays: req.EstimatedMinDays, EstimatedMaxDays: req.EstimatedMaxDays}
	f.Methods[method.ID] = method
	return method, nil
}

func (f *FakeModules) UpdateMethod(_ context.Context, id string, req shippingclient.UpdateMethodRequest) (shippingclient.ShippingMethod, error) {
	if m, ok := f.Methods[id]; ok {
		m.Name = req.Name
		m.Description = req.Description
		m.Active = req.Active
		m.EstimatedMinDays = req.EstimatedMinDays
		m.EstimatedMaxDays = req.EstimatedMaxDays
		return m, nil
	}
	return shippingclient.ShippingMethod{}, shippingclient.ErrMethodNotFound
}

func (f *FakeModules) ListZones(context.Context) ([]shippingclient.ShippingZone, error) {
	zones := make([]shippingclient.ShippingZone, 0, len(f.Zones))
	for _, z := range f.Zones {
		zones = append(zones, z)
	}
	return zones, nil
}

func (f *FakeModules) GetZone(_ context.Context, id string) (shippingclient.ShippingZone, error) {
	if z, ok := f.Zones[id]; ok {
		return z, nil
	}
	return shippingclient.ShippingZone{}, shippingclient.ErrZoneNotFound
}

func (f *FakeModules) CreateZone(_ context.Context, req shippingclient.CreateZoneRequest) (shippingclient.ShippingZone, error) {
	zone := shippingclient.ShippingZone{ID: "test-" + req.Name, Name: req.Name, Country: req.Country, State: req.State, City: req.City, PostalCodeStart: req.PostalCodeStart, PostalCodeEnd: req.PostalCodeEnd}
	f.Zones[zone.ID] = zone
	return zone, nil
}

func (f *FakeModules) UpdateZone(_ context.Context, id string, req shippingclient.UpdateZoneRequest) (shippingclient.ShippingZone, error) {
	if z, ok := f.Zones[id]; ok {
		z.Name = req.Name
		z.Country = req.Country
		z.State = req.State
		z.City = req.City
		z.PostalCodeStart = req.PostalCodeStart
		z.PostalCodeEnd = req.PostalCodeEnd
		return z, nil
	}
	return shippingclient.ShippingZone{}, shippingclient.ErrZoneNotFound
}

func (f *FakeModules) ListRates(_ context.Context, zoneID, methodID string) ([]shippingclient.ShippingRate, error) {
	rates := make([]shippingclient.ShippingRate, 0, len(f.Rates))
	for _, r := range f.Rates {
		if (zoneID == "" || r.ZoneID == zoneID) && (methodID == "" || r.MethodID == methodID) {
			rates = append(rates, r)
		}
	}
	return rates, nil
}

func (f *FakeModules) GetRate(_ context.Context, id string) (shippingclient.ShippingRate, error) {
	if r, ok := f.Rates[id]; ok {
		return r, nil
	}
	return shippingclient.ShippingRate{}, shippingclient.ErrRateNotFound
}

func (f *FakeModules) CreateRate(_ context.Context, req shippingclient.CreateRateRequest) (shippingclient.ShippingRate, error) {
	rate := shippingclient.ShippingRate{ID: "test-" + req.ZoneID + "-" + req.MethodID, ZoneID: req.ZoneID, MethodID: req.MethodID, BaseCostCents: req.BaseCostCents, FreeThresholdCents: req.FreeThresholdCents, Active: req.Active}
	f.Rates[rate.ID] = rate
	return rate, nil
}

func (f *FakeModules) UpdateRate(_ context.Context, id string, req shippingclient.UpdateRateRequest) (shippingclient.ShippingRate, error) {
	if r, ok := f.Rates[id]; ok {
		r.BaseCostCents = req.BaseCostCents
		r.FreeThresholdCents = req.FreeThresholdCents
		r.Active = req.Active
		return r, nil
	}
	return shippingclient.ShippingRate{}, shippingclient.ErrRateNotFound
}

func (f *FakeModules) GetQuote(context.Context, shippingclient.GetQuoteRequest) (shippingclient.ShippingQuote, error) {
	return shippingclient.ShippingQuote{}, shippingclient.ErrQuoteNotFound
}

func (f *FakeModules) CreateQuote(context.Context, shippingclient.CreateQuoteRequest) (shippingclient.ShippingQuote, error) {
	return shippingclient.ShippingQuote{}, nil
}

func (f *FakeModules) ValidateQuote(context.Context, shippingclient.ValidateQuoteRequest) (shippingclient.ShippingQuote, error) {
	return shippingclient.ShippingQuote{}, shippingclient.ErrQuoteNotFound
}

func (f *FakeModules) PriceQuote(context.Context, shippingclient.PriceQuoteRequest) (shippingclient.ShippingQuote, error) {
	return shippingclient.ShippingQuote{}, shippingclient.ErrQuoteNotFound
}

func (f *FakeModules) GetShipment(context.Context, string) (shippingclient.Shipment, error) {
	return shippingclient.Shipment{}, errors.New("envio não encontrado")
}

func (f *FakeModules) GetShipmentByOrderID(context.Context, string) (shippingclient.Shipment, error) {
	return shippingclient.Shipment{}, errors.New("envio não encontrado")
}

func (f *FakeModules) ListShipments(context.Context, string, int, int) ([]shippingclient.Shipment, int, error) {
	return nil, 0, nil
}

func (f *FakeModules) GetTrackingEvents(context.Context, string) ([]shippingclient.TrackingEvent, error) {
	return nil, nil
}

var _ contracts.Modules = (*FakeModules)(nil)

func CreateTestAddressForMock(fake *FakeModules, customerID, addressID string) addressclient.Address {
	addr := addressclient.Address{ID: addressID, CustomerID: customerID, Label: "Home", FullName: "John Doe", Street: "123 Main St", City: "Sao Paulo", State: "SP", Zip: "01000-000", Country: "BR", Phone: "+5511999999999", IsDefault: true, Version: 1, CreatedAt: "2024-01-01T00:00:00Z", UpdatedAt: "2024-01-01T00:00:00Z"}
	fake.Addresses[addressID] = addr
	return addr
}

func NewTestServiceWithModules() (*Service, *memory.Repository, *FakeModules) {
	return NewTestServiceWithStore(nil)
}

func NewTestServiceWithStore(idem idempotency.Store) (*Service, *memory.Repository, *FakeModules) {
	repo := memory.NewRepository()
	fakeModules := &FakeModules{
		Addresses:  make(map[string]addressclient.Address),
		Customers:  make(map[string]customerclient.Customer),
		Products:   make(map[string]catalogclient.Product),
		Categories: make(map[string]categoryclient.CategoryDTO),
		Orders:     make(map[string]orderclient.Order),
		Methods:    make(map[string]shippingclient.ShippingMethod),
		Zones:      make(map[string]shippingclient.ShippingZone),
		Rates:      make(map[string]shippingclient.ShippingRate),
	}
	svc := NewService(repo, fakeModules, idem, metrics.Noop{}, time.Hour)
	return svc, repo, fakeModules
}

func NewTestService() (*Service, *memory.Repository) {
	repo := memory.NewRepository()
	svc := NewService(repo, nil, nil, metrics.Noop{}, time.Hour)
	return svc, repo
}

func CreateTestMethod(ctx context.Context, repo *memory.Repository) domain.ShippingMethod {
	m, _ := domain.NewShippingMethod("Standard", "Standard delivery", true, 3, 5)
	_ = repo.CreateMethod(ctx, m)
	return m
}

func CreateTestZone(ctx context.Context, repo *memory.Repository) domain.ShippingZone {
	z, _ := domain.NewShippingZone("Brazil", "BR", "", "", "", "")
	_ = repo.CreateZone(ctx, z)
	return z
}

func CreateTestRate(ctx context.Context, repo *memory.Repository, zoneID, methodID string) domain.ShippingRate {
	r, _ := domain.NewShippingRate(zoneID, methodID, 599, 5000, true)
	_ = repo.CreateRate(ctx, r)
	return r
}

func CreateTestAddress(_ context.Context, _ *memory.Repository, _ string) domain.DestinationSnapshot {
	return domain.DestinationSnapshot{Label: "Home", FullName: "John Doe", Street: "123 Main St", City: "Sao Paulo", State: "SP", Zip: "01000-000", Country: "BR", Phone: "+5511999999999"}
}

// Feature-oriented helpers used by feature-mapped application tests.

func newFeatureService(t *testing.T) (*Service, *memory.Repository, *FakeModules) {
	t.Helper()
	svc, repo, fm := NewTestServiceWithStore(nil)
	return svc, repo, fm
}

func newFeatureServiceWithStore(t *testing.T, idem idempotency.Store) (*Service, *memory.Repository, *FakeModules) {
	t.Helper()
	return NewTestServiceWithStore(idem)
}

func validCreateMethodRequestFeature() CreateMethodRequest {
	return CreateMethodRequest{Name: "Standard", Description: "Standard delivery", Active: true, EstimatedMinDays: 3, EstimatedMaxDays: 5}
}

func createTestMethodFeature(t *testing.T, svc *Service, name string, active bool) domain.ShippingMethod {
	t.Helper()
	req := validCreateMethodRequestFeature()
	req.Name = name
	req.Active = active
	m, err := svc.CreateMethod(t.Context(), req)
	require.NoError(t, err)
	return m
}

func createTestZoneFeature(t *testing.T, svc *Service, name, country string) domain.ShippingZone {
	t.Helper()
	z, err := svc.CreateZone(t.Context(), CreateZoneRequest{Name: name, Country: country})
	require.NoError(t, err)
	return z
}

func createTestRateFeature(t *testing.T, svc *Service, zoneID, methodID string, baseCost, threshold int64, active bool) domain.ShippingRate {
	t.Helper()
	r, err := svc.CreateRate(t.Context(), CreateRateRequest{ZoneID: zoneID, MethodID: methodID, BaseCostCents: baseCost, FreeThresholdCents: threshold, Active: active})
	require.NoError(t, err)
	return r
}

func seedAddressFeature(fm *FakeModules, customerID, addressID, country string) {
	fm.Addresses[addressID] = addressclient.Address{ID: addressID, CustomerID: customerID, Label: "Home", FullName: "John Doe", Street: "123 Main St", City: "Sao Paulo", State: "SP", Zip: "01000-000", Country: country, Phone: "+5511999999999", IsDefault: true}
}

func seedShipmentFeature(t *testing.T, repo *memory.Repository, userID, orderID string, status domain.ShipmentStatus) domain.Shipment {
	t.Helper()
	dest := domain.DestinationSnapshot{Label: "Home", FullName: "John Doe", Street: "123 Main St", City: "Sao Paulo", State: "SP", Zip: "01000-000", Country: "BR", Phone: "+5511999999999"}
	s, err := domain.NewShipment(orderID, userID, "m1", dest, 599)
	require.NoError(t, err)
	switch status {
	case domain.ShipmentStatusProcessing:
		require.NoError(t, s.Process())
	case domain.ShipmentStatusShipped:
		require.NoError(t, s.Process())
		require.NoError(t, s.Dispatch("BR123", "Correios"))
	case domain.ShipmentStatusDelivered:
		require.NoError(t, s.Process())
		require.NoError(t, s.Dispatch("BR123", "Correios"))
		require.NoError(t, s.Deliver())
	}
	require.NoError(t, repo.CreateShipment(t.Context(), s))
	return s
}

func seedQuoteFixtureFeature(t *testing.T, repo *memory.Repository, fm *FakeModules) domain.ShippingMethod {
	t.Helper()
	m, err := domain.NewShippingMethod("Standard", "Standard delivery", true, 3, 5)
	require.NoError(t, err)
	require.NoError(t, repo.CreateMethod(t.Context(), m))
	z, err := domain.NewShippingZone("Brazil", "BR", "", "", "", "")
	require.NoError(t, err)
	require.NoError(t, repo.CreateZone(t.Context(), z))
	r, err := domain.NewShippingRate(z.ID, m.ID, 599, 5000, true)
	require.NoError(t, err)
	require.NoError(t, repo.CreateRate(t.Context(), r))
	seedAddressFeature(fm, "cust1", "addr1", "BR")
	return m
}
