package memory

import (
	"minimart/internal/shipping/domain"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMemoryRepository_Methods(t *testing.T) {
	ctx := t.Context()
	repo := NewRepository()

	m, err := domain.NewShippingMethod("Standard", "Standard delivery", true, 3, 5)
	require.NoError(t, err)
	err = repo.CreateMethod(ctx, m)
	require.NoError(t, err)

	got, err := repo.GetMethod(ctx, m.ID)
	require.NoError(t, err)
	assert.Equal(t, m.Name, got.Name)

	methods, err := repo.ListActiveMethods(ctx)
	require.NoError(t, err)
	assert.Len(t, methods, 1)

	m2, _ := domain.NewShippingMethod("Express", "Express delivery", false, 1, 2)
	_ = repo.CreateMethod(ctx, m2)
	methods, err = repo.ListActiveMethods(ctx)
	require.NoError(t, err)
	assert.Len(t, methods, 1)
}

func TestMemoryRepository_Zones(t *testing.T) {
	ctx := t.Context()
	repo := NewRepository()

	z, err := domain.NewShippingZone("Brazil", "BR", "SP", "", "", "")
	require.NoError(t, err)
	err = repo.CreateZone(ctx, z)
	require.NoError(t, err)

	got, err := repo.GetZone(ctx, z.ID)
	require.NoError(t, err)
	assert.Equal(t, z.Name, got.Name)

	zones, err := repo.ListZones(ctx)
	require.NoError(t, err)
	assert.Len(t, zones, 1)
}

func TestMemoryRepository_Rates(t *testing.T) {
	ctx := t.Context()
	repo := NewRepository()

	r, err := domain.NewShippingRate("z1", "m1", 599, 5000, true)
	require.NoError(t, err)
	err = repo.CreateRate(ctx, r)
	require.NoError(t, err)

	got, err := repo.GetRate(ctx, r.ID)
	require.NoError(t, err)
	assert.Equal(t, r.BaseCostCents, got.BaseCostCents)

	rates, err := repo.ListRates(ctx, "z1", "")
	require.NoError(t, err)
	assert.Len(t, rates, 1)

	rates, err = repo.ListRates(ctx, "", "m1")
	require.NoError(t, err)
	assert.Len(t, rates, 1)
}

func TestMemoryRepository_Quotes(t *testing.T) {
	ctx := t.Context()
	repo := NewRepository()

	dest := domain.DestinationSnapshot{FullName: "John", Street: "123 St", City: "City", Zip: "12345", Country: "BR"}
	q, err := domain.NewShippingQuote("cust1", "m1", "addr1", "key-1", dest, 599, 1200, 5000, 3, 5)
	require.NoError(t, err)
	err = repo.CreateQuote(ctx, q)
	require.NoError(t, err)

	got, err := repo.GetQuote(ctx, q.ID)
	require.NoError(t, err)
	assert.Equal(t, q.ShippingCostCents, got.ShippingCostCents)
	assert.Equal(t, q.SubtotalCents, got.SubtotalCents)
	assert.Equal(t, q.FreeThresholdCents, got.FreeThresholdCents)

	q.ShippingCostCents = 0
	q.SubtotalCents = 6000
	require.NoError(t, repo.UpdateQuote(ctx, q))

	got, err = repo.GetQuote(ctx, q.ID)
	require.NoError(t, err)
	assert.Equal(t, int64(0), got.ShippingCostCents)
	assert.Equal(t, int64(6000), got.SubtotalCents)
}

func TestMemoryRepository_QuoteIdempotencyKey(t *testing.T) {
	ctx := t.Context()
	repo := NewRepository()

	dest := domain.DestinationSnapshot{FullName: "John", Street: "123 St", City: "City", Zip: "12345", Country: "BR"}
	q, err := domain.NewShippingQuote("cust1", "m1", "addr1", "key-1", dest, 599, 1200, 5000, 3, 5)
	require.NoError(t, err)
	require.NoError(t, repo.CreateQuote(ctx, q))

	got, err := repo.GetQuoteByIdempotencyKey(ctx, "key-1")
	require.NoError(t, err)
	assert.Equal(t, q.ID, got.ID)

	_, err = repo.GetQuoteByIdempotencyKey(ctx, "missing")
	require.ErrorIs(t, err, domain.ErrQuoteNotFound)

	// A second quote may not take the same key.
	other, err := domain.NewShippingQuote("cust1", "m1", "addr1", "key-1", dest, 599, 1200, 5000, 3, 5)
	require.NoError(t, err)
	require.ErrorIs(t, repo.CreateQuote(ctx, other), domain.ErrIdempotencyConflict)

	// A quote without a key never conflicts.
	noKey, err := domain.NewShippingQuote("cust1", "m1", "addr1", "", dest, 599, 1200, 5000, 3, 5)
	require.NoError(t, err)
	require.NoError(t, repo.CreateQuote(ctx, noKey))
}

func TestMemoryRepository_ExpireQuotes(t *testing.T) {
	ctx := t.Context()
	repo := NewRepository()

	dest := domain.DestinationSnapshot{FullName: "John", Street: "123 St", City: "City", Zip: "12345", Country: "BR"}
	q, _ := domain.NewShippingQuote("cust1", "m1", "addr1", "", dest, 599, 0, 0, 3, 5)
	q.ExpiresAt = time.Now().UTC().Add(-time.Hour)
	_ = repo.CreateQuote(ctx, q)

	err := repo.ExpireQuotes(ctx)
	require.NoError(t, err)

	got, err := repo.GetQuote(ctx, q.ID)
	require.NoError(t, err)
	assert.Equal(t, domain.QuoteStatusExpired, got.Status)
}

func TestMemoryRepository_Shipments(t *testing.T) {
	ctx := t.Context()
	repo := NewRepository()

	dest := domain.DestinationSnapshot{FullName: "John", Street: "123 St", City: "City", Zip: "12345", Country: "BR"}
	s, err := domain.NewShipment("order1", "user1", "m1", dest, 599)
	require.NoError(t, err)
	err = repo.CreateShipment(ctx, s)
	require.NoError(t, err)

	got, err := repo.GetShipment(ctx, s.ID)
	require.NoError(t, err)
	assert.Equal(t, s.OrderID, got.OrderID)

	got, err = repo.GetShipmentByOrderID(ctx, "order1")
	require.NoError(t, err)
	assert.Equal(t, s.ID, got.ID)

	// Duplicate order_id should fail
	s2, _ := domain.NewShipment("order1", "user2", "m1", dest, 599)
	err = repo.CreateShipment(ctx, s2)
	require.Error(t, err)

	shipments, total, err := repo.ListShipments(ctx, "user1", 1, 20)
	require.NoError(t, err)
	assert.Len(t, shipments, 1)
	assert.Equal(t, 1, total)
}

func TestMemoryRepository_TrackingEvents(t *testing.T) {
	ctx := t.Context()
	repo := NewRepository()

	e, err := domain.NewTrackingEvent("ship1", domain.ShipmentStatusShipped, "Dispatched", "Sao Paulo", time.Now())
	require.NoError(t, err)
	err = repo.AddTrackingEvent(ctx, e)
	require.NoError(t, err)

	events, err := repo.GetTrackingEvents(ctx, "ship1")
	require.NoError(t, err)
	assert.Len(t, events, 1)
	assert.Equal(t, e.Description, events[0].Description)
}
