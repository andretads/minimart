package module

import (
	"context"
	"errors"
	"minimart/internal/shipping/domain"
	"minimart/pkg/module/contracts"

	addressclient "minimart/internal/address/presentation/module/client"
	catalogclient "minimart/internal/catalog/presentation/module/client"
	categoryclient "minimart/internal/category/presentation/module/client"
	customerclient "minimart/internal/customer/presentation/module/client"
	orderclient "minimart/internal/order/presentation/module/client"
	shippingclient "minimart/internal/shipping/presentation/module/client"
)

type testModules struct{}

func (t testModules) GetAddress(ctx context.Context, req addressclient.GetAddressRequest) (addressclient.Address, error) {
	return addressclient.Address{}, addressclient.ErrAddressNotFound
}

func (t testModules) GetDefaultAddressByCustomerID(ctx context.Context, customerID string) (addressclient.Address, error) {
	return addressclient.Address{}, addressclient.ErrAddressNotFound
}

func (t testModules) GetProduct(ctx context.Context, id string) (catalogclient.Product, error) {
	return catalogclient.Product{}, catalogclient.ErrProductNotFound
}

func (t testModules) ReserveStock(ctx context.Context, orderID string, items []catalogclient.LineItem) error {
	return nil
}

func (t testModules) ReleaseStock(ctx context.Context, orderID string) error {
	return nil
}

func (t testModules) EnsureCustomer(ctx context.Context, userID string) (customerclient.Customer, error) {
	return customerclient.Customer{}, nil
}

func (t testModules) GetCustomerByID(ctx context.Context, id string) (customerclient.Customer, error) {
	return customerclient.Customer{}, nil
}

func (t testModules) GetCustomerByUserID(ctx context.Context, userID string) (customerclient.Customer, error) {
	return customerclient.Customer{}, nil
}

func (t testModules) GetCategory(ctx context.Context, id string) (categoryclient.CategoryDTO, error) {
	return categoryclient.CategoryDTO{}, nil
}

func (t testModules) GetOrder(ctx context.Context, id string) (orderclient.Order, error) {
	return orderclient.Order{}, nil
}

func (t testModules) PlaceOrder(ctx context.Context, req orderclient.PlaceOrderRequest) (orderclient.Order, error) {
	return orderclient.Order{}, nil
}

func (t testModules) ListMethods(ctx context.Context) ([]shippingclient.ShippingMethod, error) {
	return nil, nil
}

func (t testModules) GetMethod(ctx context.Context, id string) (shippingclient.ShippingMethod, error) {
	return shippingclient.ShippingMethod{}, shippingclient.ErrMethodNotFound
}

func (t testModules) CreateMethod(ctx context.Context, req shippingclient.CreateMethodRequest) (shippingclient.ShippingMethod, error) {
	return shippingclient.ShippingMethod{}, nil
}

func (t testModules) UpdateMethod(ctx context.Context, id string, req shippingclient.UpdateMethodRequest) (shippingclient.ShippingMethod, error) {
	return shippingclient.ShippingMethod{}, nil
}

func (t testModules) ListZones(ctx context.Context) ([]shippingclient.ShippingZone, error) {
	return nil, nil
}

func (t testModules) GetZone(ctx context.Context, id string) (shippingclient.ShippingZone, error) {
	return shippingclient.ShippingZone{}, shippingclient.ErrZoneNotFound
}

func (t testModules) CreateZone(ctx context.Context, req shippingclient.CreateZoneRequest) (shippingclient.ShippingZone, error) {
	return shippingclient.ShippingZone{}, nil
}

func (t testModules) UpdateZone(ctx context.Context, id string, req shippingclient.UpdateZoneRequest) (shippingclient.ShippingZone, error) {
	return shippingclient.ShippingZone{}, nil
}

func (t testModules) ListRates(ctx context.Context, zoneID, methodID string) ([]shippingclient.ShippingRate, error) {
	return nil, nil
}

func (t testModules) GetRate(ctx context.Context, id string) (shippingclient.ShippingRate, error) {
	return shippingclient.ShippingRate{}, shippingclient.ErrRateNotFound
}

func (t testModules) CreateRate(ctx context.Context, req shippingclient.CreateRateRequest) (shippingclient.ShippingRate, error) {
	return shippingclient.ShippingRate{}, nil
}

func (t testModules) UpdateRate(ctx context.Context, id string, req shippingclient.UpdateRateRequest) (shippingclient.ShippingRate, error) {
	return shippingclient.ShippingRate{}, nil
}

func (t testModules) GetQuote(ctx context.Context, req shippingclient.GetQuoteRequest) (shippingclient.ShippingQuote, error) {
	return shippingclient.ShippingQuote{}, shippingclient.ErrQuoteNotFound
}

func (t testModules) CreateQuote(ctx context.Context, req shippingclient.CreateQuoteRequest) (shippingclient.ShippingQuote, error) {
	return shippingclient.ShippingQuote{}, nil
}

func (t testModules) ValidateQuote(ctx context.Context, req shippingclient.ValidateQuoteRequest) (shippingclient.ShippingQuote, error) {
	return shippingclient.ShippingQuote{}, shippingclient.ErrQuoteNotFound
}

func (t testModules) PriceQuote(ctx context.Context, req shippingclient.PriceQuoteRequest) (shippingclient.ShippingQuote, error) {
	return shippingclient.ShippingQuote{}, shippingclient.ErrQuoteNotFound
}

func (t testModules) GetShipment(ctx context.Context, id string) (shippingclient.Shipment, error) {
	return shippingclient.Shipment{}, errors.New("envio não encontrado")
}

func (t testModules) GetShipmentByOrderID(ctx context.Context, orderID string) (shippingclient.Shipment, error) {
	return shippingclient.Shipment{}, errors.New("envio não encontrado")
}

func (t testModules) ListShipments(ctx context.Context, userID string, page, pageSize int) ([]shippingclient.Shipment, int, error) {
	return nil, 0, nil
}

func (t testModules) GetTrackingEvents(ctx context.Context, shipmentID string) ([]shippingclient.TrackingEvent, error) {
	return nil, nil
}

var _ contracts.Modules = (*testModules)(nil)

type failingRepo struct {
	err error
}

func (f failingRepo) GetMethod(ctx context.Context, id string) (domain.ShippingMethod, error) {
	return domain.ShippingMethod{}, f.err
}

func (f failingRepo) ListActiveMethods(ctx context.Context) ([]domain.ShippingMethod, error) {
	return nil, nil
}

func (f failingRepo) ListAllMethods(ctx context.Context) ([]domain.ShippingMethod, error) {
	return nil, nil
}

func (f failingRepo) CreateMethod(ctx context.Context, method domain.ShippingMethod) error {
	return nil
}

func (f failingRepo) UpdateMethod(ctx context.Context, method domain.ShippingMethod) error {
	return nil
}

func (f failingRepo) GetZone(ctx context.Context, id string) (domain.ShippingZone, error) {
	return domain.ShippingZone{}, nil
}

func (f failingRepo) ListZones(ctx context.Context) ([]domain.ShippingZone, error)   { return nil, nil }
func (f failingRepo) CreateZone(ctx context.Context, zone domain.ShippingZone) error { return nil }
func (f failingRepo) UpdateZone(ctx context.Context, zone domain.ShippingZone) error { return nil }
func (f failingRepo) GetRate(ctx context.Context, id string) (domain.ShippingRate, error) {
	return domain.ShippingRate{}, nil
}

func (f failingRepo) ListRates(ctx context.Context, zoneID, methodID string) ([]domain.ShippingRate, error) {
	return nil, nil
}

func (f failingRepo) ListRatesByZone(ctx context.Context, zoneID string) ([]domain.ShippingRate, error) {
	return nil, nil
}

func (f failingRepo) ListRatesByMethod(ctx context.Context, methodID string) ([]domain.ShippingRate, error) {
	return nil, nil
}
func (f failingRepo) CreateRate(ctx context.Context, rate domain.ShippingRate) error { return nil }
func (f failingRepo) UpdateRate(ctx context.Context, rate domain.ShippingRate) error { return nil }
func (f failingRepo) GetQuote(ctx context.Context, id string) (domain.ShippingQuote, error) {
	return domain.ShippingQuote{}, nil
}

func (f failingRepo) GetQuoteByIdempotencyKey(ctx context.Context, key string) (domain.ShippingQuote, error) {
	return domain.ShippingQuote{}, nil
}

func (f failingRepo) CreateQuote(ctx context.Context, quote domain.ShippingQuote) error { return nil }

func (f failingRepo) UpdateQuote(ctx context.Context, quote domain.ShippingQuote) error { return nil }

func (f failingRepo) ExpireQuotes(ctx context.Context) error { return nil }

func (f failingRepo) GetShipment(ctx context.Context, id string) (domain.Shipment, error) {
	return domain.Shipment{}, nil
}

func (f failingRepo) GetShipmentByOrderID(ctx context.Context, orderID string) (domain.Shipment, error) {
	return domain.Shipment{}, nil
}

func (f failingRepo) ListShipments(ctx context.Context, userID string, page, pageSize int) ([]domain.Shipment, int, error) {
	return nil, 0, nil
}

func (f failingRepo) CreateShipment(ctx context.Context, shipment domain.Shipment) error { return nil }

func (f failingRepo) CreateShipmentForQuote(ctx context.Context, shipment domain.Shipment, quote domain.ShippingQuote) error {
	return nil
}

func (f failingRepo) UpdateShipment(ctx context.Context, shipment domain.Shipment) error { return nil }

func (f failingRepo) AddTrackingEvent(ctx context.Context, event domain.TrackingEvent) error {
	return nil
}

func (f failingRepo) GetTrackingEvents(ctx context.Context, shipmentID string) ([]domain.TrackingEvent, error) {
	return nil, nil
}
