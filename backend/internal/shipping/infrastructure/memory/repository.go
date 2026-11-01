package memory

import (
	"context"
	"errors"
	"minimart/internal/shipping/domain"
	"sort"
	"sync"
	"time"
)

var _ domain.Repository = (*Repository)(nil)

type Repository struct {
	mu               sync.RWMutex
	methods          map[string]domain.ShippingMethod
	zones            map[string]domain.ShippingZone
	rates            map[string]domain.ShippingRate
	quotes           map[string]domain.ShippingQuote
	quotesByKey      map[string]string
	shipments        map[string]domain.Shipment
	shipmentsByOrder map[string]string
	trackingEvents   map[string][]domain.TrackingEvent
}

func NewRepository() *Repository {
	return &Repository{
		methods:          make(map[string]domain.ShippingMethod),
		zones:            make(map[string]domain.ShippingZone),
		rates:            make(map[string]domain.ShippingRate),
		quotes:           make(map[string]domain.ShippingQuote),
		quotesByKey:      make(map[string]string),
		shipments:        make(map[string]domain.Shipment),
		shipmentsByOrder: make(map[string]string),
		trackingEvents:   make(map[string][]domain.TrackingEvent),
	}
}

// Shipping Methods
func (r *Repository) GetMethod(ctx context.Context, id string) (domain.ShippingMethod, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	m, ok := r.methods[id]
	if !ok {
		return domain.ShippingMethod{}, domain.ErrMethodNotFound
	}
	return m, nil
}

func (r *Repository) ListActiveMethods(ctx context.Context) ([]domain.ShippingMethod, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	var methods []domain.ShippingMethod
	for _, m := range r.methods {
		if m.Active {
			methods = append(methods, m)
		}
	}
	sort.SliceStable(methods, func(i, j int) bool {
		return methods[i].Name < methods[j].Name
	})
	return methods, nil
}

func (r *Repository) ListAllMethods(ctx context.Context) ([]domain.ShippingMethod, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	var methods []domain.ShippingMethod
	for _, m := range r.methods {
		methods = append(methods, m)
	}
	sort.SliceStable(methods, func(i, j int) bool {
		return methods[i].Name < methods[j].Name
	})
	return methods, nil
}

func (r *Repository) CreateMethod(ctx context.Context, method domain.ShippingMethod) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.methods[method.ID]; ok {
		return errors.New("method already exists")
	}
	r.methods[method.ID] = method
	return nil
}

func (r *Repository) UpdateMethod(ctx context.Context, method domain.ShippingMethod) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.methods[method.ID]; !ok {
		return domain.ErrMethodNotFound
	}
	r.methods[method.ID] = method
	return nil
}

// Shipping Zones
func (r *Repository) GetZone(ctx context.Context, id string) (domain.ShippingZone, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	z, ok := r.zones[id]
	if !ok {
		return domain.ShippingZone{}, domain.ErrZoneNotFound
	}
	return z, nil
}

func (r *Repository) ListZones(ctx context.Context) ([]domain.ShippingZone, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	var zones []domain.ShippingZone
	for _, z := range r.zones {
		zones = append(zones, z)
	}
	sort.SliceStable(zones, func(i, j int) bool {
		return zones[i].Name < zones[j].Name
	})
	return zones, nil
}

func (r *Repository) CreateZone(ctx context.Context, zone domain.ShippingZone) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.zones[zone.ID]; ok {
		return errors.New("zone already exists")
	}
	r.zones[zone.ID] = zone
	return nil
}

func (r *Repository) UpdateZone(ctx context.Context, zone domain.ShippingZone) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.zones[zone.ID]; !ok {
		return domain.ErrZoneNotFound
	}
	r.zones[zone.ID] = zone
	return nil
}

// Shipping Rates
func (r *Repository) GetRate(ctx context.Context, id string) (domain.ShippingRate, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	rate, ok := r.rates[id]
	if !ok {
		return domain.ShippingRate{}, domain.ErrRateNotFound
	}
	return rate, nil
}

func (r *Repository) ListRates(ctx context.Context, zoneID, methodID string) ([]domain.ShippingRate, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	var rates []domain.ShippingRate
	for _, rate := range r.rates {
		if zoneID != "" && rate.ZoneID != zoneID {
			continue
		}
		if methodID != "" && rate.MethodID != methodID {
			continue
		}
		rates = append(rates, rate)
	}
	sort.SliceStable(rates, func(i, j int) bool {
		if rates[i].ZoneID != rates[j].ZoneID {
			return rates[i].ZoneID < rates[j].ZoneID
		}
		if rates[i].MethodID != rates[j].MethodID {
			return rates[i].MethodID < rates[j].MethodID
		}
		if rates[i].BaseCostCents != rates[j].BaseCostCents {
			return rates[i].BaseCostCents < rates[j].BaseCostCents
		}
		return rates[i].ID < rates[j].ID
	})
	return rates, nil
}

func (r *Repository) ListRatesByZone(ctx context.Context, zoneID string) ([]domain.ShippingRate, error) {
	return r.ListRates(ctx, zoneID, "")
}

func (r *Repository) ListRatesByMethod(ctx context.Context, methodID string) ([]domain.ShippingRate, error) {
	return r.ListRates(ctx, "", methodID)
}

func (r *Repository) CreateRate(ctx context.Context, rate domain.ShippingRate) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.rates[rate.ID]; ok {
		return errors.New("rate already exists")
	}
	r.rates[rate.ID] = rate
	return nil
}

func (r *Repository) UpdateRate(ctx context.Context, rate domain.ShippingRate) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.rates[rate.ID]; !ok {
		return domain.ErrRateNotFound
	}
	r.rates[rate.ID] = rate
	return nil
}

// Shipping Quotes
func (r *Repository) GetQuote(ctx context.Context, id string) (domain.ShippingQuote, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	q, ok := r.quotes[id]
	if !ok {
		return domain.ShippingQuote{}, domain.ErrQuoteNotFound
	}
	return q, nil
}

// GetQuoteByIdempotencyKey returns the quote created under an idempotency key,
// or ErrQuoteNotFound when none carries it.
func (r *Repository) GetQuoteByIdempotencyKey(ctx context.Context, key string) (domain.ShippingQuote, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	id, ok := r.quotesByKey[key]
	if !ok {
		return domain.ShippingQuote{}, domain.ErrQuoteNotFound
	}
	q, ok := r.quotes[id]
	if !ok {
		return domain.ShippingQuote{}, domain.ErrQuoteNotFound
	}
	return q, nil
}

func (r *Repository) CreateQuote(ctx context.Context, quote domain.ShippingQuote) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.quotes[quote.ID]; ok {
		return errors.New("quote already exists")
	}
	if quote.IdempotencyKey != "" {
		if _, ok := r.quotesByKey[quote.IdempotencyKey]; ok {
			return domain.ErrIdempotencyConflict
		}
	}
	r.quotes[quote.ID] = quote
	if quote.IdempotencyKey != "" {
		r.quotesByKey[quote.IdempotencyKey] = quote.ID
	}
	return nil
}

func (r *Repository) UpdateQuote(ctx context.Context, quote domain.ShippingQuote) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.quotes[quote.ID]; !ok {
		return domain.ErrQuoteNotFound
	}
	r.quotes[quote.ID] = quote
	if quote.IdempotencyKey != "" {
		r.quotesByKey[quote.IdempotencyKey] = quote.ID
	}
	return nil
}

func (r *Repository) ExpireQuotes(ctx context.Context) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	now := time.Now().UTC()
	for id, q := range r.quotes {
		if q.Status == domain.QuoteStatusActive && now.After(q.ExpiresAt) {
			q.Status = domain.QuoteStatusExpired
			r.quotes[id] = q
		}
	}
	return nil
}

// Shipments
func (r *Repository) GetShipment(ctx context.Context, id string) (domain.Shipment, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	s, ok := r.shipments[id]
	if !ok {
		return domain.Shipment{}, domain.ErrShipmentNotFound
	}
	return s, nil
}

func (r *Repository) GetShipmentByOrderID(ctx context.Context, orderID string) (domain.Shipment, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	shipmentID, ok := r.shipmentsByOrder[orderID]
	if !ok {
		return domain.Shipment{}, domain.ErrShipmentNotFound
	}
	s, ok := r.shipments[shipmentID]
	if !ok {
		return domain.Shipment{}, domain.ErrShipmentNotFound
	}
	return s, nil
}

func (r *Repository) ListShipments(ctx context.Context, userID string, page, pageSize int) ([]domain.Shipment, int, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	var shipments []domain.Shipment
	for _, s := range r.shipments {
		if s.UserID == userID {
			shipments = append(shipments, s)
		}
	}
	sort.SliceStable(shipments, func(i, j int) bool {
		if shipments[i].CreatedAt.Equal(shipments[j].CreatedAt) {
			return shipments[i].ID < shipments[j].ID
		}
		return shipments[i].CreatedAt.After(shipments[j].CreatedAt)
	})
	total := len(shipments)
	start := (page - 1) * pageSize
	if start >= total {
		return []domain.Shipment{}, total, nil
	}
	end := min(start+pageSize, total)
	return shipments[start:end], total, nil
}

func (r *Repository) CreateShipment(ctx context.Context, shipment domain.Shipment) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.shipmentsByOrder[shipment.OrderID]; ok {
		return domain.ErrShipmentExists
	}
	if _, ok := r.shipments[shipment.ID]; ok {
		return errors.New("shipment already exists")
	}
	r.shipments[shipment.ID] = shipment
	r.shipmentsByOrder[shipment.OrderID] = shipment.ID
	return nil
}

// CreateShipmentForQuote creates the shipment and consumes the quote together,
// mirroring the Postgres adapter's single transaction.
func (r *Repository) CreateShipmentForQuote(ctx context.Context, shipment domain.Shipment, quote domain.ShippingQuote) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.shipmentsByOrder[shipment.OrderID]; ok {
		return domain.ErrShipmentExists
	}
	if _, ok := r.shipments[shipment.ID]; ok {
		return errors.New("shipment already exists")
	}
	stored, ok := r.quotes[quote.ID]
	if !ok {
		return domain.ErrQuoteNotFound
	}
	if stored.Status != domain.QuoteStatusActive {
		return domain.ErrQuoteConsumed
	}
	stored.Status = domain.QuoteStatusConsumed
	r.quotes[quote.ID] = stored
	r.shipments[shipment.ID] = shipment
	r.shipmentsByOrder[shipment.OrderID] = shipment.ID
	return nil
}

func (r *Repository) UpdateShipment(ctx context.Context, shipment domain.Shipment) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.shipments[shipment.ID]; !ok {
		return domain.ErrShipmentNotFound
	}
	r.shipments[shipment.ID] = shipment
	return nil
}

// Tracking Events
func (r *Repository) AddTrackingEvent(ctx context.Context, event domain.TrackingEvent) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.trackingEvents[event.ShipmentID] = append(r.trackingEvents[event.ShipmentID], event)
	return nil
}

func (r *Repository) GetTrackingEvents(ctx context.Context, shipmentID string) ([]domain.TrackingEvent, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	events := r.trackingEvents[shipmentID]
	// Return events in reverse chronological order (newest first)
	result := make([]domain.TrackingEvent, len(events))
	for i, e := range events {
		result[len(events)-1-i] = e
	}
	return result, nil
}
