//go:build integration

package test

import (
	"minimart/internal/shipping/application"
	"minimart/internal/shipping/domain"
	idemredis "minimart/pkg/idempotency/redis"
	"minimart/pkg/metrics"
	"minimart/pkg/redis"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestShippingIdempotency_RedeliveredOrderPaidReplaysShipment proves the
// idempotency store remembers the shipment created for an order, so a
// redelivered event replays it instead of creating a second one.
func TestShippingIdempotency_RedeliveredOrderPaidReplaysShipment(t *testing.T) {
	repo, pool := newShippingRepo(t)
	orderID := uniqueShippingID()
	cleanup(t, pool, orderID)

	fixture := seedShippingFixture(t, repo, "it-cust")
	cleanup(t, pool, fixture.ids()...)

	method, err := repo.GetMethod(t.Context(), fixture.Method.ID)
	require.NoError(t, err)

	rdb, err := redis.New(testRedisURL)
	require.NoError(t, err)
	t.Cleanup(func() { _ = rdb.Close() })

	store := idemredis.New(rdb.Client)
	svc := application.NewService(repo, nil, store, metrics.Noop{}, time.Hour)

	first, err := svc.CreateShipmentFromOrderPaid(t.Context(), orderID, "it-user", "it-cust", fixture.Quote.ID, method.ID, fixture.Quote.DestinationSnapshot, 1500)
	require.NoError(t, err)

	second, err := svc.CreateShipmentFromOrderPaid(t.Context(), orderID, "it-user", "it-cust", fixture.Quote.ID, method.ID, fixture.Quote.DestinationSnapshot, 1500)
	require.NoError(t, err)
	assert.Equal(t, first.ID, second.ID)

	ids := shipmentIDs(t, pool, orderID)
	assert.Len(t, ids, 1, "a redelivered event must not create a second shipment")

	// The store remembers the shipment under the order's key.
	rec, found, err := store.Get(t.Context(), "shipping:shipment:"+orderID)
	require.NoError(t, err)
	require.True(t, found)
	assert.Equal(t, first.ID, rec.ResourceID)
}

// TestShippingIdempotency_StoreUnavailableFallsBackToDatabase proves the
// database's unique index still prevents a duplicate when Redis is down.
func TestShippingIdempotency_StoreUnavailableFallsBackToDatabase(t *testing.T) {
	repo, pool := newShippingRepo(t)
	orderID := uniqueShippingID()
	cleanup(t, pool, orderID)

	fixture := seedShippingFixture(t, repo, "it-cust")
	cleanup(t, pool, fixture.ids()...)

	method, err := repo.GetMethod(t.Context(), fixture.Method.ID)
	require.NoError(t, err)

	svc := application.NewService(repo, nil, failingIdem{}, metrics.Noop{}, time.Hour)

	first, err := svc.CreateShipmentFromOrderPaid(t.Context(), orderID, "it-user", "it-cust", fixture.Quote.ID, method.ID, fixture.Quote.DestinationSnapshot, 1500)
	require.NoError(t, err)

	second, err := svc.CreateShipmentFromOrderPaid(t.Context(), orderID, "it-user", "it-cust", fixture.Quote.ID, method.ID, fixture.Quote.DestinationSnapshot, 1500)
	require.NoError(t, err)
	assert.Equal(t, first.ID, second.ID)

	ids := shipmentIDs(t, pool, orderID)
	assert.Len(t, ids, 1, "the database must reject a duplicate even without Redis")
}

// TestShippingIdempotency_KeyReuseWithDifferentPayloadIsAConflict proves a
// changed payload under the same key is reported instead of silently ignored.
func TestShippingIdempotency_KeyReuseWithDifferentPayloadIsAConflict(t *testing.T) {
	repo, pool := newShippingRepo(t)
	orderID := uniqueShippingID()
	cleanup(t, pool, orderID)

	fixture := seedShippingFixture(t, repo, "it-cust")
	cleanup(t, pool, fixture.ids()...)

	method, err := repo.GetMethod(t.Context(), fixture.Method.ID)
	require.NoError(t, err)

	rdb, err := redis.New(testRedisURL)
	require.NoError(t, err)
	t.Cleanup(func() { _ = rdb.Close() })

	store := idemredis.New(rdb.Client)
	svc := application.NewService(repo, nil, store, metrics.Noop{}, time.Hour)

	_, err = svc.CreateShipmentFromOrderPaid(t.Context(), orderID, "it-user", "it-cust", fixture.Quote.ID, method.ID, fixture.Quote.DestinationSnapshot, 1500)
	require.NoError(t, err)

	_, err = svc.CreateShipmentFromOrderPaid(t.Context(), orderID, "it-user", "it-cust", fixture.Quote.ID, method.ID, fixture.Quote.DestinationSnapshot, 2500)
	require.ErrorIs(t, err, domain.ErrIdempotencyConflict)
}

// TestShippingIdempotency_QuoteIsConsumedWithTheShipment proves the quote is
// spent in the same transaction as the shipment, so it cannot be reused.
func TestShippingIdempotency_QuoteIsConsumedWithTheShipment(t *testing.T) {
	repo, pool := newShippingRepo(t)
	orderID := uniqueShippingID()
	cleanup(t, pool, orderID)

	fixture := seedShippingFixture(t, repo, "it-cust")
	cleanup(t, pool, fixture.ids()...)

	method, err := repo.GetMethod(t.Context(), fixture.Method.ID)
	require.NoError(t, err)

	svc := application.NewService(repo, nil, nil, metrics.Noop{}, time.Hour)

	_, err = svc.CreateShipmentFromOrderPaid(t.Context(), orderID, "it-user", "it-cust", fixture.Quote.ID, method.ID, fixture.Quote.DestinationSnapshot, 1500)
	require.NoError(t, err)

	consumed, err := repo.GetQuote(t.Context(), fixture.Quote.ID)
	require.NoError(t, err)
	assert.Equal(t, domain.QuoteStatusConsumed, consumed.Status)

	// The quote cannot be spent by another order.
	otherOrderID := uniqueShippingID()
	cleanup(t, pool, otherOrderID)
	_, err = svc.CreateShipmentFromOrderPaid(t.Context(), otherOrderID, "it-user", "it-cust", fixture.Quote.ID, method.ID, fixture.Quote.DestinationSnapshot, 1500)
	require.ErrorIs(t, err, domain.ErrQuoteConsumed)
}

// TestShippingIdempotency_ReservationIsReleasedOnFailure proves a failed
// creation frees the reservation so a later redelivery can proceed.
func TestShippingIdempotency_ReservationIsReleasedOnFailure(t *testing.T) {
	repo, pool := newShippingRepo(t)
	orderID := uniqueShippingID()
	cleanup(t, pool, orderID)

	fixture := seedShippingFixture(t, repo, "it-cust")
	cleanup(t, pool, fixture.ids()...)

	method, err := repo.GetMethod(t.Context(), fixture.Method.ID)
	require.NoError(t, err)

	rdb, err := redis.New(testRedisURL)
	require.NoError(t, err)
	t.Cleanup(func() { _ = rdb.Close() })

	store := idemredis.New(rdb.Client)
	svc := application.NewService(repo, nil, store, metrics.Noop{}, time.Hour)

	// The quote does not belong to the customer, so the creation fails.
	_, err = svc.CreateShipmentFromOrderPaid(t.Context(), orderID, "it-user", "other-cust", fixture.Quote.ID, method.ID, fixture.Quote.DestinationSnapshot, 1500)
	require.ErrorIs(t, err, domain.ErrOwnershipMismatch)

	_, found, err := store.Get(t.Context(), "shipping:shipment:"+orderID)
	require.NoError(t, err)
	assert.False(t, found, "a failed creation must release its reservation")

	// The key is free, so a correct redelivery succeeds.
	shipment, err := svc.CreateShipmentFromOrderPaid(t.Context(), orderID, "it-user", "it-cust", fixture.Quote.ID, method.ID, fixture.Quote.DestinationSnapshot, 1500)
	require.NoError(t, err)
	assert.NotEmpty(t, shipment.ID)
}
