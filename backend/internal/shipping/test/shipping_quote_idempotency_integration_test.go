//go:build integration

package test

import (
	"context"
	"minimart/internal/address/presentation/module/client"
	"minimart/internal/shipping/application"
	"minimart/internal/shipping/domain"
	"minimart/internal/shipping/infrastructure/postgres"
	idemredis "minimart/pkg/idempotency/redis"
	"minimart/pkg/metrics"
	"minimart/pkg/module/contracts"
	"minimart/pkg/redis"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fakeModules serves the address a quote needs and leaves every other
// contract unimplemented.
type fakeModules struct {
	contracts.Modules
	address client.Address
}

func (f fakeModules) GetAddress(_ context.Context, req client.GetAddressRequest) (client.Address, error) {
	if req.ID != f.address.ID {
		return client.Address{}, client.ErrAddressNotFound
	}
	return f.address, nil
}

// quoteFixture is the reference data a quote needs: a method, a zone and a
// rate for one address.
type quoteFixture struct {
	Method domain.ShippingMethod
	Rate   domain.ShippingRate
	Zone   domain.ShippingZone
	Addr   client.Address
}

func (f quoteFixture) ids() []string {
	return []string{f.Method.ID, f.Rate.ID, f.Zone.ID}
}

// seedQuoteFixture creates the method, zone, rate and address a quote needs.
// The zone covers a country unique to the test, so only this fixture's zone
// matches the address even when other tests share the database.
func seedQuoteFixture(t *testing.T, repo *postgres.Repository, customerID string) quoteFixture {
	t.Helper()
	ctx := t.Context()
	suffix := uniqueShippingID()
	country := "C-" + suffix

	method, err := domain.NewShippingMethod("Standard "+suffix, "Standard delivery", true, 3, 5)
	require.NoError(t, err)
	require.NoError(t, repo.CreateMethod(ctx, method))

	zone, err := domain.NewShippingZone("Zone "+suffix, country, "", "", "", "")
	require.NoError(t, err)
	require.NoError(t, repo.CreateZone(ctx, zone))

	rate, err := domain.NewShippingRate(zone.ID, method.ID, 599, 5000, true)
	require.NoError(t, err)
	require.NoError(t, repo.CreateRate(ctx, rate))

	return quoteFixture{
		Method: method,
		Rate:   rate,
		Zone:   zone,
		Addr: client.Address{
			ID:         "addr-" + suffix,
			CustomerID: customerID,
			Label:      "Home",
			FullName:   "John Doe",
			Street:     "123 Main St",
			City:       "Test City",
			State:      "TS",
			Zip:        "01000-000",
			Country:    country,
			Phone:      "+5511999999999",
		},
	}
}

// TestShippingIdempotency_QuoteIsReplayedForTheSameKey proves the store
// remembers the quote created for a key, so a retry replays it instead of
// creating a second one.
func TestShippingIdempotency_QuoteIsReplayedForTheSameKey(t *testing.T) {
	repo, pool := newShippingRepo(t)
	fixture := seedQuoteFixture(t, repo, "it-cust")
	cleanup(t, pool, fixture.ids()...)
	key := "key-" + uniqueShippingID()

	rdb, err := redis.New(testRedisURL)
	require.NoError(t, err)
	t.Cleanup(func() { _ = rdb.Close() })

	store := idemredis.New(rdb.Client)
	svc := application.NewService(repo, fakeModules{address: fixture.Addr}, store, metrics.Noop{}, time.Hour)

	req := application.CreateQuoteRequest{
		CustomerID:     "it-cust",
		MethodID:       fixture.Method.ID,
		AddressID:      fixture.Addr.ID,
		SubtotalCents:  1200,
		IdempotencyKey: key,
	}

	first, err := svc.CreateQuote(t.Context(), req)
	require.NoError(t, err)
	cleanup(t, pool, first.ID)

	second, err := svc.CreateQuote(t.Context(), req)
	require.NoError(t, err)
	assert.Equal(t, first.ID, second.ID)

	ids := quoteIDs(t, pool, "shipping:quote:"+key)
	assert.Len(t, ids, 1, "a retried request must not create a second quote")

	// The store remembers the quote under the request's key.
	rec, found, err := store.Get(t.Context(), "shipping:quote:"+key)
	require.NoError(t, err)
	require.True(t, found)
	assert.Equal(t, first.ID, rec.ResourceID)
}

// TestShippingIdempotency_QuoteStoreUnavailableFallsBackToDatabase proves the
// database's unique index still prevents a duplicate when Redis is down.
func TestShippingIdempotency_QuoteStoreUnavailableFallsBackToDatabase(t *testing.T) {
	repo, pool := newShippingRepo(t)
	fixture := seedQuoteFixture(t, repo, "it-cust")
	cleanup(t, pool, fixture.ids()...)
	key := "key-" + uniqueShippingID()

	svc := application.NewService(repo, fakeModules{address: fixture.Addr}, failingIdem{}, metrics.Noop{}, time.Hour)

	req := application.CreateQuoteRequest{
		CustomerID:     "it-cust",
		MethodID:       fixture.Method.ID,
		AddressID:      fixture.Addr.ID,
		SubtotalCents:  1200,
		IdempotencyKey: key,
	}

	first, err := svc.CreateQuote(t.Context(), req)
	require.NoError(t, err)
	cleanup(t, pool, first.ID)

	second, err := svc.CreateQuote(t.Context(), req)
	require.NoError(t, err)
	assert.Equal(t, first.ID, second.ID)

	ids := quoteIDs(t, pool, "shipping:quote:"+key)
	assert.Len(t, ids, 1, "the database must reject a duplicate even without Redis")
}

// TestShippingIdempotency_QuoteKeyReuseWithOtherParametersIsAConflict proves a
// changed request under the same key is reported instead of silently ignored.
func TestShippingIdempotency_QuoteKeyReuseWithOtherParametersIsAConflict(t *testing.T) {
	repo, pool := newShippingRepo(t)
	fixture := seedQuoteFixture(t, repo, "it-cust")
	cleanup(t, pool, fixture.ids()...)
	key := "key-" + uniqueShippingID()

	rdb, err := redis.New(testRedisURL)
	require.NoError(t, err)
	t.Cleanup(func() { _ = rdb.Close() })

	store := idemredis.New(rdb.Client)
	svc := application.NewService(repo, fakeModules{address: fixture.Addr}, store, metrics.Noop{}, time.Hour)

	first, err := svc.CreateQuote(t.Context(), application.CreateQuoteRequest{
		CustomerID:     "it-cust",
		MethodID:       fixture.Method.ID,
		AddressID:      fixture.Addr.ID,
		SubtotalCents:  1200,
		IdempotencyKey: key,
	})
	require.NoError(t, err)
	cleanup(t, pool, first.ID)

	_, err = svc.CreateQuote(t.Context(), application.CreateQuoteRequest{
		CustomerID:     "it-cust",
		MethodID:       fixture.Method.ID,
		AddressID:      fixture.Addr.ID,
		SubtotalCents:  2400,
		IdempotencyKey: key,
	})
	require.ErrorIs(t, err, domain.ErrIdempotencyConflict)
}

// TestShippingIdempotency_QuoteRequiresAnIdempotencyKey proves a quote is
// never created without a key to make its retry safe.
func TestShippingIdempotency_QuoteRequiresAnIdempotencyKey(t *testing.T) {
	repo, pool := newShippingRepo(t)
	fixture := seedQuoteFixture(t, repo, "it-cust")
	cleanup(t, pool, fixture.ids()...)

	svc := application.NewService(repo, fakeModules{address: fixture.Addr}, nil, metrics.Noop{}, time.Hour)

	_, err := svc.CreateQuote(t.Context(), application.CreateQuoteRequest{
		CustomerID: "it-cust",
		MethodID:   fixture.Method.ID,
		AddressID:  fixture.Addr.ID,
	})
	require.ErrorIs(t, err, domain.ErrIdempotencyKeyRequired)
}

// quoteIDs returns the ids of the quotes recorded for an idempotency key.
func quoteIDs(t *testing.T, pool *pgxpool.Pool, key string) []string {
	t.Helper()
	rows, err := pool.Query(t.Context(), "SELECT id FROM shipping_quotes WHERE idempotency_key = $1", key)
	require.NoError(t, err)
	defer rows.Close()

	var ids []string
	for rows.Next() {
		var id string
		require.NoError(t, rows.Scan(&id))
		ids = append(ids, id)
	}
	require.NoError(t, rows.Err())
	return ids
}
