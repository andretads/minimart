//go:build integration

package test

import (
	"context"
	"errors"
	"fmt"
	"minimart/internal/shipping/domain"
	shippingoutbox "minimart/internal/shipping/infrastructure/outbox"
	"minimart/internal/shipping/infrastructure/postgres"
	"minimart/pkg/idempotency"
	"minimart/pkg/kafka"
	outboxpkg "minimart/pkg/outbox"
	outboxpg "minimart/pkg/outbox/postgres"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"
)

// uniqueShippingID returns a value unlikely to collide with a previous run's
// rows, used for the ids of everything a test creates.
func uniqueShippingID() string {
	return fmt.Sprintf("it-ship-%d", time.Now().UnixNano())
}

// newShippingRepo connects to the real database and returns the adapter. The
// read and write pools point at the same database, and the repository writes
// domain events to the shared outbox via the real Publisher.
func newShippingRepo(t *testing.T) (*postgres.Repository, *pgxpool.Pool) {
	t.Helper()

	pool, err := pgxpool.New(t.Context(), testDSN)
	require.NoError(t, err)
	t.Cleanup(pool.Close)

	store := outboxpg.NewStore(pool)
	return postgres.NewRepository(pool, pool, outboxpkg.NewPublisher(store)), pool
}

// cleanup removes the rows a test created. t.Context() is cancelled before
// cleanup runs, so use a fresh context.
func cleanup(t *testing.T, pool *pgxpool.Pool, ids ...string) {
	t.Helper()
	t.Cleanup(func() {
		//nolint:usetesting // t.Context() is cancelled before cleanup runs
		ctx := context.Background()
		for _, id := range ids {
			_, _ = pool.Exec(ctx, "DELETE FROM outbox WHERE message_key = $1", id)
			_, _ = pool.Exec(ctx, "DELETE FROM shipment_tracking_events WHERE shipment_id = $1", id)
			_, _ = pool.Exec(ctx, "DELETE FROM shipments WHERE id = $1 OR order_id = $1", id)
			_, _ = pool.Exec(ctx, "DELETE FROM shipping_quotes WHERE id = $1", id)
			_, _ = pool.Exec(ctx, "DELETE FROM shipping_rates WHERE id = $1", id)
			_, _ = pool.Exec(ctx, "DELETE FROM shipping_zones WHERE id = $1", id)
			_, _ = pool.Exec(ctx, "DELETE FROM shipping_methods WHERE id = $1", id)
		}
	})
}

// newShippingRelay builds the shipping outbox relay over the real shared
// outbox store with fast intervals suited to a test.
func newShippingRelay(t *testing.T, pool *pgxpool.Pool, publisher kafka.Publisher) *shippingoutbox.Relay {
	t.Helper()
	return shippingoutbox.NewRelay(outboxpg.NewStore(pool), publisher, "it-relay", shippingoutbox.Settings{
		ProcessInterval:           50 * time.Millisecond,
		LockCheckerInterval:       time.Hour,
		CleanupWorkerInterval:     time.Hour,
		MaxLockTimeDuration:       time.Minute,
		MessagesRetentionDuration: time.Hour,
		MaxSendAttempts:           3,
	})
}

// shippingFixture is the reference data a paid order needs: a shipping method,
// a zone, a rate and an active quote.
type shippingFixture struct {
	Quote   domain.ShippingQuote
	Method  domain.ShippingMethod
	Rate    domain.ShippingRate
	Zone    domain.ShippingZone
	OrderID string
}

// ids returns every id the fixture created, so cleanup can remove them.
func (f shippingFixture) ids() []string {
	return []string{f.Quote.ID, f.Method.ID, f.Rate.ID, f.Zone.ID}
}

// seedShippingFixture creates the method, zone, rate and active quote a paid
// order needs. The names carry the unique suffix so repeated runs never
// collide on the tables' unique name columns.
func seedShippingFixture(t *testing.T, repo *postgres.Repository, customerID string) shippingFixture {
	t.Helper()
	ctx := t.Context()
	suffix := uniqueShippingID()

	method, err := domain.NewShippingMethod("Standard "+suffix, "Standard delivery", true, 3, 5)
	require.NoError(t, err)
	require.NoError(t, repo.CreateMethod(ctx, method))

	zone, err := domain.NewShippingZone("Brazil "+suffix, "BR", "", "", "", "")
	require.NoError(t, err)
	require.NoError(t, repo.CreateZone(ctx, zone))

	rate, err := domain.NewShippingRate(zone.ID, method.ID, 599, 5000, true)
	require.NoError(t, err)
	require.NoError(t, repo.CreateRate(ctx, rate))

	quote, err := domain.NewShippingQuote(customerID, method.ID, "addr1", "", domain.DestinationSnapshot{
		Label: "Home", FullName: "John Doe", Street: "123 Main St", City: "Sao Paulo",
		State: "SP", Zip: "01000-000", Country: "BR", Phone: "+5511999999999",
	}, 599, 1200, 5000, 3, 5)
	require.NoError(t, err)
	require.NoError(t, repo.CreateQuote(ctx, quote))

	return shippingFixture{Quote: quote, Method: method, Rate: rate, Zone: zone}
}

// outboxRecord is one row of the shared outbox table.
type outboxRecord struct {
	Topic string
	Body  []byte
	State int
}

// outboxRows returns the outbox rows recorded for a message key, oldest first.
func outboxRows(t *testing.T, pool *pgxpool.Pool, key string) []outboxRecord {
	t.Helper()
	rows, err := pool.Query(t.Context(),
		"SELECT topic, body, state FROM outbox WHERE message_key = $1 ORDER BY created_on", key)
	require.NoError(t, err)
	defer rows.Close()

	var records []outboxRecord
	for rows.Next() {
		var rec outboxRecord
		require.NoError(t, rows.Scan(&rec.Topic, &rec.Body, &rec.State))
		records = append(records, rec)
	}
	require.NoError(t, rows.Err())
	return records
}

// shipmentIDs returns the ids of the shipments recorded for an order.
func shipmentIDs(t *testing.T, pool *pgxpool.Pool, orderID string) []string {
	t.Helper()
	rows, err := pool.Query(t.Context(), "SELECT id FROM shipments WHERE order_id = $1", orderID)
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

// failingIdem simulates an unavailable idempotency store: every operation
// returns an error so the service's fail-open path falls back to the database.
type failingIdem struct{}

func (failingIdem) Get(context.Context, string) (idempotency.Record, bool, error) {
	return idempotency.Record{}, false, errors.New("idempotency store unavailable")
}

func (failingIdem) Reserve(context.Context, string, string, time.Duration) (bool, error) {
	return false, errors.New("idempotency store unavailable")
}

func (failingIdem) Complete(context.Context, string, string, string, time.Duration) error {
	return errors.New("idempotency store unavailable")
}

func (failingIdem) Release(context.Context, string, string) error {
	return errors.New("idempotency store unavailable")
}
