//go:build integration

package test

import (
	"context"
	"fmt"
	addressapp "minimart/internal/address/application"
	addressdomain "minimart/internal/address/domain"
	addresspg "minimart/internal/address/infrastructure/postgres"
	addressclient "minimart/internal/address/presentation/module/client"
	catalogclient "minimart/internal/catalog/presentation/module/client"
	categoryclient "minimart/internal/category/presentation/module/client"
	customerapp "minimart/internal/customer/application"
	customerdomain "minimart/internal/customer/domain"
	customerpg "minimart/internal/customer/infrastructure/postgres"
	customerclient "minimart/internal/customer/presentation/module/client"
	orderclient "minimart/internal/order/presentation/module/client"
	shippingapp "minimart/internal/shipping/application"
	"minimart/internal/shipping/domain"
	"minimart/internal/shipping/infrastructure/postgres"
	shippingclient "minimart/internal/shipping/presentation/module/client"
	"minimart/pkg/idempotency"
	"minimart/pkg/metrics"
	outboxpkg "minimart/pkg/outbox"
	outboxpg "minimart/pkg/outbox/postgres"
	"sync/atomic"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"
)

// uniqueShippingSeq disambiguates ids generated within the same clock tick.
var uniqueShippingSeq atomic.Int64

// uniqueShippingID returns a value unlikely to collide with a previous run's
// rows, used for the ids of everything a test creates.
func uniqueShippingID() string {
	return fmt.Sprintf("it-ship-%d-%d", time.Now().UnixNano(), uniqueShippingSeq.Add(1))
}

type featureModules struct {
	addrSvc *addressapp.Service
}

func (m featureModules) GetAddress(ctx context.Context, req addressclient.GetAddressRequest) (addressclient.Address, error) {
	a, err := m.addrSvc.GetAddress(ctx, req.ID, req.CustomerID)
	if err != nil {
		return addressclient.Address{}, addressclient.ErrAddressNotFound
	}
	return addressclient.Address{ID: a.ID, CustomerID: a.CustomerID, Label: a.Label, FullName: a.FullName, Street: a.Street, City: a.City, State: a.State, Zip: a.Zip, Country: a.Country, Phone: a.Phone, IsDefault: a.IsDefault}, nil
}

func (m featureModules) GetDefaultAddressByCustomerID(context.Context, string) (addressclient.Address, error) {
	return addressclient.Address{}, addressclient.ErrAddressNotFound
}

func (m featureModules) GetProduct(context.Context, string) (catalogclient.Product, error) {
	return catalogclient.Product{}, nil
}

func (m featureModules) ReserveStock(context.Context, string, []catalogclient.LineItem) error {
	return nil
}
func (m featureModules) ReleaseStock(context.Context, string) error { return nil }
func (m featureModules) EnsureCustomer(context.Context, string) (customerclient.Customer, error) {
	return customerclient.Customer{}, nil
}

func (m featureModules) GetCustomerByID(context.Context, string) (customerclient.Customer, error) {
	return customerclient.Customer{}, nil
}

func (m featureModules) GetCustomerByUserID(context.Context, string) (customerclient.Customer, error) {
	return customerclient.Customer{}, nil
}

func (m featureModules) GetCategory(context.Context, string) (categoryclient.CategoryDTO, error) {
	return categoryclient.CategoryDTO{}, nil
}

func (m featureModules) GetOrder(context.Context, string) (orderclient.Order, error) {
	return orderclient.Order{}, nil
}

func (m featureModules) PlaceOrder(context.Context, orderclient.PlaceOrderRequest) (orderclient.Order, error) {
	return orderclient.Order{}, nil
}

func (m featureModules) ListMethods(context.Context) ([]shippingclient.ShippingMethod, error) {
	return nil, nil
}

func (m featureModules) GetMethod(context.Context, string) (shippingclient.ShippingMethod, error) {
	return shippingclient.ShippingMethod{}, nil
}

func (m featureModules) CreateMethod(context.Context, shippingclient.CreateMethodRequest) (shippingclient.ShippingMethod, error) {
	return shippingclient.ShippingMethod{}, nil
}

func (m featureModules) UpdateMethod(context.Context, string, shippingclient.UpdateMethodRequest) (shippingclient.ShippingMethod, error) {
	return shippingclient.ShippingMethod{}, nil
}

func (m featureModules) ListZones(context.Context) ([]shippingclient.ShippingZone, error) {
	return nil, nil
}

func (m featureModules) GetZone(context.Context, string) (shippingclient.ShippingZone, error) {
	return shippingclient.ShippingZone{}, nil
}

func (m featureModules) CreateZone(context.Context, shippingclient.CreateZoneRequest) (shippingclient.ShippingZone, error) {
	return shippingclient.ShippingZone{}, nil
}

func (m featureModules) UpdateZone(context.Context, string, shippingclient.UpdateZoneRequest) (shippingclient.ShippingZone, error) {
	return shippingclient.ShippingZone{}, nil
}

func (m featureModules) ListRates(context.Context, string, string) ([]shippingclient.ShippingRate, error) {
	return nil, nil
}

func (m featureModules) GetRate(context.Context, string) (shippingclient.ShippingRate, error) {
	return shippingclient.ShippingRate{}, nil
}

func (m featureModules) CreateRate(context.Context, shippingclient.CreateRateRequest) (shippingclient.ShippingRate, error) {
	return shippingclient.ShippingRate{}, nil
}

func (m featureModules) UpdateRate(context.Context, string, shippingclient.UpdateRateRequest) (shippingclient.ShippingRate, error) {
	return shippingclient.ShippingRate{}, nil
}

func (m featureModules) GetQuote(context.Context, shippingclient.GetQuoteRequest) (shippingclient.ShippingQuote, error) {
	return shippingclient.ShippingQuote{}, nil
}

func (m featureModules) CreateQuote(context.Context, shippingclient.CreateQuoteRequest) (shippingclient.ShippingQuote, error) {
	return shippingclient.ShippingQuote{}, nil
}

func (m featureModules) ValidateQuote(context.Context, shippingclient.ValidateQuoteRequest) (shippingclient.ShippingQuote, error) {
	return shippingclient.ShippingQuote{}, nil
}

func (m featureModules) PriceQuote(context.Context, shippingclient.PriceQuoteRequest) (shippingclient.ShippingQuote, error) {
	return shippingclient.ShippingQuote{}, nil
}

func (m featureModules) GetShipment(context.Context, string) (shippingclient.Shipment, error) {
	return shippingclient.Shipment{}, nil
}

func (m featureModules) GetShipmentByOrderID(context.Context, string) (shippingclient.Shipment, error) {
	return shippingclient.Shipment{}, nil
}

func (m featureModules) ListShipments(context.Context, string, int, int) ([]shippingclient.Shipment, int, error) {
	return nil, 0, nil
}

func (m featureModules) GetTrackingEvents(context.Context, string) ([]shippingclient.TrackingEvent, error) {
	return nil, nil
}

func newFeatureServices(t *testing.T) (*shippingapp.Service, *addressapp.Service, *customerapp.Service, *postgres.Repository, *pgxpool.Pool) {
	t.Helper()
	return newFeatureServicesWithIdem(t, nil)
}

func newFeatureServicesWithIdem(t *testing.T, idem idempotency.Store) (*shippingapp.Service, *addressapp.Service, *customerapp.Service, *postgres.Repository, *pgxpool.Pool) {
	t.Helper()
	pool, err := pgxpool.New(t.Context(), testDSN)
	require.NoError(t, err)
	t.Cleanup(pool.Close)

	store := outboxpg.NewStore(pool)
	shipRepo := postgres.NewRepository(pool, pool, outboxpkg.NewPublisher(store))
	addrRepo := addresspg.NewAddressRepository(pool, pool)
	custRepo := customerpg.NewCustomerRepository(pool, pool)

	addrSvc := addressapp.NewService(addrRepo, metrics.Noop{})
	custSvc := customerapp.NewService(custRepo, metrics.Noop{})
	shipSvc := shippingapp.NewService(shipRepo, featureModules{addrSvc: addrSvc}, idem, metrics.Noop{}, time.Hour)
	return shipSvc, addrSvc, custSvc, shipRepo, pool
}

func createTestCustomer(t *testing.T, svc *customerapp.Service) customerdomain.Customer {
	t.Helper()
	c, err := svc.CreateCustomer(t.Context(), customerapp.CreateCustomerRequest{UserID: "user-" + uniqueShippingID(), DisplayName: "Test User"})
	require.NoError(t, err)
	return c
}

func createTestAddressExact(t *testing.T, svc *addressapp.Service, customerID, country, city, state, zip string) addressdomain.Address {
	t.Helper()
	a, err := svc.CreateAddress(t.Context(), addressapp.CreateAddressRequest{CustomerID: customerID, Label: "Home", FullName: "John Doe", Street: "123 Main St", City: city, State: state, Zip: zip, Country: country, Phone: "+5511999999999"})
	require.NoError(t, err)
	return a
}

func mustCreateMethod(t *testing.T, svc *shippingapp.Service, name string, active bool) domain.ShippingMethod {
	t.Helper()
	m, err := svc.CreateMethod(t.Context(), shippingapp.CreateMethodRequest{Name: name + "-" + uniqueShippingID(), Description: "Standard delivery", Active: active, EstimatedMinDays: 3, EstimatedMaxDays: 5})
	require.NoError(t, err)
	return m
}

func mustCreateZone(t *testing.T, svc *shippingapp.Service, name, country string) domain.ShippingZone {
	t.Helper()
	z, err := svc.CreateZone(t.Context(), shippingapp.CreateZoneRequest{Name: name + "-" + uniqueShippingID(), Country: country})
	require.NoError(t, err)
	return z
}

func mustCreateRate(t *testing.T, svc *shippingapp.Service, zoneID, methodID string, baseCost, threshold int64, active bool) domain.ShippingRate {
	t.Helper()
	r, err := svc.CreateRate(t.Context(), shippingapp.CreateRateRequest{ZoneID: zoneID, MethodID: methodID, BaseCostCents: baseCost, FreeThresholdCents: threshold, Active: active})
	require.NoError(t, err)
	return r
}

func seedStoredShipment(t *testing.T, svc *shippingapp.Service, repo *postgres.Repository, userID, orderID string, status domain.ShipmentStatus) domain.Shipment {
	t.Helper()
	ctx := t.Context()
	m := mustCreateMethod(t, svc, "SeedMethod", true)
	dest := domain.DestinationSnapshot{Label: "Home", FullName: "John Doe", Street: "123 Main St", City: "Sao Paulo", State: "SP", Zip: "01000-000", Country: "BR", Phone: "+5511999999999"}
	s, err := domain.NewShipment(orderID, userID, m.ID, dest, 599)
	require.NoError(t, err)
	require.NoError(t, repo.CreateShipment(ctx, s))
	loaded, err := repo.GetShipment(ctx, s.ID)
	require.NoError(t, err)
	switch status {
	case domain.ShipmentStatusProcessing:
		require.NoError(t, loaded.Process())
		require.NoError(t, repo.UpdateShipment(ctx, loaded))
	case domain.ShipmentStatusShipped:
		require.NoError(t, loaded.Process())
		require.NoError(t, loaded.Dispatch("BR123", "Correios"))
		require.NoError(t, repo.UpdateShipment(ctx, loaded))
	case domain.ShipmentStatusDelivered:
		require.NoError(t, loaded.Process())
		require.NoError(t, loaded.Dispatch("BR123", "Correios"))
		require.NoError(t, loaded.Deliver())
		require.NoError(t, repo.UpdateShipment(ctx, loaded))
	}
	final, err := repo.GetShipment(ctx, s.ID)
	require.NoError(t, err)
	return final
}

func mustCreateZoneExact(t *testing.T, svc *shippingapp.Service, req shippingapp.CreateZoneRequest) domain.ShippingZone {
	t.Helper()
	z, err := svc.CreateZone(t.Context(), req)
	require.NoError(t, err)
	return z
}
