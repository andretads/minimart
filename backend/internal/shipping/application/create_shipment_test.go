package application

import (
	"context"
	"errors"
	"minimart/internal/shipping/domain"
	"minimart/internal/shipping/infrastructure/memory"
	"minimart/pkg/idempotency"
	"minimart/pkg/metrics"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// memIdempotency is a minimal in-memory idempotency.Store for tests.
type memIdempotency struct {
	records    map[string]idempotency.Record
	getErr     error
	reserveErr error
}

var _ idempotency.Store = (*memIdempotency)(nil)

func newMemIdempotency() *memIdempotency {
	return &memIdempotency{records: map[string]idempotency.Record{}}
}

func (s *memIdempotency) Get(_ context.Context, key string) (idempotency.Record, bool, error) {
	if s.getErr != nil {
		return idempotency.Record{}, false, s.getErr
	}
	r, ok := s.records[key]
	return r, ok, nil
}

func (s *memIdempotency) Reserve(_ context.Context, key, fp string, _ time.Duration) (bool, error) {
	if s.reserveErr != nil {
		return false, s.reserveErr
	}
	if _, ok := s.records[key]; ok {
		return false, nil
	}
	s.records[key] = idempotency.Record{Fingerprint: fp}
	return true, nil
}

func (s *memIdempotency) Complete(_ context.Context, key, fp, resourceID string, _ time.Duration) error {
	s.records[key] = idempotency.Record{Fingerprint: fp, ResourceID: resourceID}
	return nil
}

func (s *memIdempotency) Release(_ context.Context, key, _ string) error {
	delete(s.records, key)
	return nil
}

// seedShipping builds the method, zone, rate and active quote a paid order
// needs, and returns the quote.
func seedShipping(t *testing.T, repo *memory.Repository, customerID string) domain.ShippingQuote {
	t.Helper()
	ctx := t.Context()

	method, err := domain.NewShippingMethod("Standard", "Standard delivery", true, 3, 5)
	require.NoError(t, err)
	require.NoError(t, repo.CreateMethod(ctx, method))

	zone, err := domain.NewShippingZone("Brazil", "BR", "", "", "", "")
	require.NoError(t, err)
	require.NoError(t, repo.CreateZone(ctx, zone))

	rate, err := domain.NewShippingRate(zone.ID, method.ID, 599, 5000, true)
	require.NoError(t, err)
	require.NoError(t, repo.CreateRate(ctx, rate))

	quote, err := domain.NewShippingQuote(customerID, method.ID, "addr1", "", domain.DestinationSnapshot{
		FullName: "John", Street: "123 St", City: "City", Zip: "12345", Country: "BR",
	}, 599, 0, 0, 3, 5)
	require.NoError(t, err)
	require.NoError(t, repo.CreateQuote(ctx, quote))

	return quote
}

func TestService_CreateShipment(t *testing.T) {
	ctx := t.Context()
	repo := memory.NewRepository()
	svc := NewService(repo, nil, nil, metrics.Noop{}, time.Hour)

	dest := domain.DestinationSnapshot{FullName: "John", Street: "123 St", City: "City", Zip: "12345", Country: "BR"}

	shipment, err := svc.CreateShipment(ctx, CreateShipmentRequest{
		OrderID:             "order1",
		UserID:              "user1",
		MethodID:            "method1",
		DestinationSnapshot: dest,
		ShippingCostCents:   599,
	})
	require.NoError(t, err)
	assert.Equal(t, "order1", shipment.OrderID)
	assert.Equal(t, "user1", shipment.UserID)
	assert.Equal(t, "method1", shipment.MethodID)
	assert.Equal(t, int64(599), shipment.ShippingCostCents)
	assert.Equal(t, domain.ShipmentStatusPending, shipment.Status)
}

func TestService_CreateShipment_DuplicateOrder(t *testing.T) {
	ctx := t.Context()
	repo := memory.NewRepository()
	svc := NewService(repo, nil, nil, metrics.Noop{}, time.Hour)

	dest := domain.DestinationSnapshot{FullName: "John", Street: "123 St", City: "City", Zip: "12345", Country: "BR"}

	_, err := svc.CreateShipment(ctx, CreateShipmentRequest{
		OrderID:             "order1",
		UserID:              "user1",
		MethodID:            "method1",
		DestinationSnapshot: dest,
		ShippingCostCents:   599,
	})
	require.NoError(t, err)

	_, err = svc.CreateShipment(ctx, CreateShipmentRequest{
		OrderID:             "order1",
		UserID:              "user2",
		MethodID:            "method1",
		DestinationSnapshot: dest,
		ShippingCostCents:   599,
	})
	require.ErrorIs(t, err, domain.ErrShipmentExists)
}

func TestService_CreateShipmentFromOrderPaid(t *testing.T) {
	ctx := t.Context()
	repo := memory.NewRepository()
	svc := NewService(repo, nil, nil, metrics.Noop{}, time.Hour)

	quote := seedShipping(t, repo, "cust1")
	method, err := repo.GetMethod(ctx, quote.MethodID)
	require.NoError(t, err)

	shipment, err := svc.CreateShipmentFromOrderPaid(ctx, "order1", "user1", "cust1", quote.ID, method.ID, quote.DestinationSnapshot, 599)
	require.NoError(t, err)
	assert.Equal(t, "order1", shipment.OrderID)
	assert.Equal(t, "user1", shipment.UserID)
	assert.Equal(t, method.ID, shipment.MethodID)
	assert.Equal(t, int64(599), shipment.ShippingCostCents)

	// The quote is consumed in the same transaction as the shipment.
	consumed, err := repo.GetQuote(ctx, quote.ID)
	require.NoError(t, err)
	assert.Equal(t, domain.QuoteStatusConsumed, consumed.Status)
}

func TestService_CreateShipmentFromOrderPaid_Duplicate(t *testing.T) {
	ctx := t.Context()
	repo := memory.NewRepository()
	svc := NewService(repo, nil, nil, metrics.Noop{}, time.Hour)

	quote := seedShipping(t, repo, "cust1")
	method, err := repo.GetMethod(ctx, quote.MethodID)
	require.NoError(t, err)

	shipment1, err := svc.CreateShipmentFromOrderPaid(ctx, "order1", "user1", "cust1", quote.ID, method.ID, quote.DestinationSnapshot, 599)
	require.NoError(t, err)

	// Try to create again - should return existing
	shipment2, err := svc.CreateShipmentFromOrderPaid(ctx, "order1", "user1", "cust1", quote.ID, method.ID, quote.DestinationSnapshot, 599)
	require.NoError(t, err)
	assert.Equal(t, shipment1.ID, shipment2.ID) // Should return same shipment
}

func TestService_CreateShipmentFromOrderPaid_IdempotentReplay(t *testing.T) {
	ctx := t.Context()
	repo := memory.NewRepository()
	idem := newMemIdempotency()
	svc := NewService(repo, nil, idem, metrics.Noop{}, time.Hour)

	quote := seedShipping(t, repo, "cust1")
	method, err := repo.GetMethod(ctx, quote.MethodID)
	require.NoError(t, err)

	first, err := svc.CreateShipmentFromOrderPaid(ctx, "order1", "user1", "cust1", quote.ID, method.ID, quote.DestinationSnapshot, 599)
	require.NoError(t, err)

	// A redelivered order-paid event replays the stored shipment.
	second, err := svc.CreateShipmentFromOrderPaid(ctx, "order1", "user1", "cust1", quote.ID, method.ID, quote.DestinationSnapshot, 599)
	require.NoError(t, err)
	assert.Equal(t, first.ID, second.ID)

	rec, found, err := idem.Get(ctx, shipmentKey("order1"))
	require.NoError(t, err)
	require.True(t, found)
	assert.Equal(t, first.ID, rec.ResourceID)
}

func TestService_CreateShipmentFromOrderPaid_StoreUnavailableFallsBackToDatabase(t *testing.T) {
	ctx := t.Context()
	repo := memory.NewRepository()
	idem := newMemIdempotency()
	idem.getErr = errors.New("redis down")
	svc := NewService(repo, nil, idem, metrics.Noop{}, time.Hour)

	quote := seedShipping(t, repo, "cust1")
	method, err := repo.GetMethod(ctx, quote.MethodID)
	require.NoError(t, err)

	first, err := svc.CreateShipmentFromOrderPaid(ctx, "order1", "user1", "cust1", quote.ID, method.ID, quote.DestinationSnapshot, 599)
	require.NoError(t, err)

	// With the store unavailable the database's unique index still prevents a
	// second shipment for the same order.
	second, err := svc.CreateShipmentFromOrderPaid(ctx, "order1", "user1", "cust1", quote.ID, method.ID, quote.DestinationSnapshot, 599)
	require.NoError(t, err)
	assert.Equal(t, first.ID, second.ID)
}

func TestService_CreateShipmentFromOrderPaid_FingerprintConflict(t *testing.T) {
	ctx := t.Context()
	repo := memory.NewRepository()
	idem := newMemIdempotency()
	svc := NewService(repo, nil, idem, metrics.Noop{}, time.Hour)

	quote := seedShipping(t, repo, "cust1")
	method, err := repo.GetMethod(ctx, quote.MethodID)
	require.NoError(t, err)

	_, err = svc.CreateShipmentFromOrderPaid(ctx, "order1", "user1", "cust1", quote.ID, method.ID, quote.DestinationSnapshot, 599)
	require.NoError(t, err)

	// The same key with different parameters is a conflict.
	_, err = svc.CreateShipmentFromOrderPaid(ctx, "order1", "user1", "cust1", quote.ID, method.ID, quote.DestinationSnapshot, 999)
	require.ErrorIs(t, err, domain.ErrIdempotencyConflict)
}

func TestService_CreateShipmentFromOrderPaid_QuoteNotFound(t *testing.T) {
	ctx := t.Context()
	repo := memory.NewRepository()
	svc := NewService(repo, nil, nil, metrics.Noop{}, time.Hour)

	_, err := svc.CreateShipmentFromOrderPaid(ctx, "order1", "user1", "cust1", "missing-quote", "method1", domain.DestinationSnapshot{}, 599)
	require.ErrorIs(t, err, domain.ErrQuoteNotFound)
}
