//go:build integration

package test

import (
	"minimart/internal/shipping/domain"
	"minimart/pkg/kafka"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestShippingRepository_CreateShipmentWritesEventToOutbox proves the shipment
// row and its shipment-created event are written in the same transaction: the
// outbox row exists as soon as the shipment does.
func TestShippingRepository_CreateShipmentWritesEventToOutbox(t *testing.T) {
	repo, pool := newShippingRepo(t)
	ctx := t.Context()
	orderID := uniqueShippingID()
	cleanup(t, pool, orderID)

	fixture := seedShippingFixture(t, repo, "it-cust")
	cleanup(t, pool, fixture.ids()...)

	shipment, err := domain.NewShipment(orderID, "it-user", fixture.Method.ID, fixture.Quote.DestinationSnapshot, 599)
	require.NoError(t, err)
	require.NoError(t, repo.CreateShipment(ctx, shipment))

	got, err := repo.GetShipmentByOrderID(ctx, orderID)
	require.NoError(t, err)
	assert.Equal(t, shipment.ID, got.ID)
	assert.Equal(t, domain.ShipmentStatusPending, got.Status)
	assert.Equal(t, fixture.Quote.DestinationSnapshot, got.DestinationSnapshot)

	records := outboxRows(t, pool, shipment.ID)
	require.Len(t, records, 1, "the shipment-created event must be recorded in the outbox")
	assert.Equal(t, kafka.ShipmentCreatedTopic.String(), records[0].Topic)
}

// TestShippingRepository_UpdateShipmentWritesEventToOutbox proves a shipment
// transition and its event share one transaction.
func TestShippingRepository_UpdateShipmentWritesEventToOutbox(t *testing.T) {
	repo, pool := newShippingRepo(t)
	ctx := t.Context()
	orderID := uniqueShippingID()
	cleanup(t, pool, orderID)

	fixture := seedShippingFixture(t, repo, "it-cust")
	cleanup(t, pool, fixture.ids()...)

	shipment, err := domain.NewShipment(orderID, "it-user", fixture.Method.ID, fixture.Quote.DestinationSnapshot, 599)
	require.NoError(t, err)
	require.NoError(t, repo.CreateShipment(ctx, shipment))

	// The use cases load the shipment, transition it and persist it again, so
	// only the transition's event is written.
	loaded, err := repo.GetShipment(ctx, shipment.ID)
	require.NoError(t, err)
	require.NoError(t, loaded.Process())
	require.NoError(t, loaded.Dispatch("TRACK123", "Correios"))
	require.NoError(t, repo.UpdateShipment(ctx, loaded))

	got, err := repo.GetShipment(ctx, shipment.ID)
	require.NoError(t, err)
	assert.Equal(t, domain.ShipmentStatusShipped, got.Status)
	assert.Equal(t, "TRACK123", got.TrackingCode)

	records := outboxRows(t, pool, shipment.ID)
	require.Len(t, records, 2, "the created and the shipped event must both be recorded")
	assert.Equal(t, kafka.ShipmentCreatedTopic.String(), records[0].Topic)
	assert.Equal(t, kafka.ShipmentShippedTopic.String(), records[1].Topic)
}

// TestShippingRepository_CreateShipmentForQuoteConsumesQuote proves the
// shipment, the consumed quote and the event commit together.
func TestShippingRepository_CreateShipmentForQuoteConsumesQuote(t *testing.T) {
	repo, pool := newShippingRepo(t)
	ctx := t.Context()
	orderID := uniqueShippingID()
	cleanup(t, pool, orderID)

	fixture := seedShippingFixture(t, repo, "it-cust")
	cleanup(t, pool, fixture.ids()...)

	shipment, err := domain.NewShipment(orderID, "it-user", fixture.Method.ID, fixture.Quote.DestinationSnapshot, 599)
	require.NoError(t, err)
	require.NoError(t, fixture.Quote.Consume())
	require.NoError(t, repo.CreateShipmentForQuote(ctx, shipment, fixture.Quote))

	consumed, err := repo.GetQuote(ctx, fixture.Quote.ID)
	require.NoError(t, err)
	assert.Equal(t, domain.QuoteStatusConsumed, consumed.Status)

	records := outboxRows(t, pool, shipment.ID)
	require.Len(t, records, 1)
	assert.Equal(t, kafka.ShipmentCreatedTopic.String(), records[0].Topic)
}

// TestShippingRepository_UniqueOrderID proves the database rejects a second
// shipment for the same order and the adapter reports the domain sentinel.
func TestShippingRepository_UniqueOrderID(t *testing.T) {
	repo, pool := newShippingRepo(t)
	ctx := t.Context()
	orderID := uniqueShippingID()
	cleanup(t, pool, orderID)

	fixture := seedShippingFixture(t, repo, "it-cust")
	cleanup(t, pool, fixture.ids()...)

	first, err := domain.NewShipment(orderID, "it-user", fixture.Method.ID, fixture.Quote.DestinationSnapshot, 599)
	require.NoError(t, err)
	require.NoError(t, repo.CreateShipment(ctx, first))

	second, err := domain.NewShipment(orderID, "it-user", fixture.Method.ID, fixture.Quote.DestinationSnapshot, 599)
	require.NoError(t, err)
	require.ErrorIs(t, repo.CreateShipment(ctx, second), domain.ErrShipmentExists)
}

// TestShippingRepository_ConsumedQuoteCannotBeSpentTwice proves the quote is
// spent exactly once: a second shipment for another order cannot reuse it.
func TestShippingRepository_ConsumedQuoteCannotBeSpentTwice(t *testing.T) {
	repo, pool := newShippingRepo(t)
	ctx := t.Context()
	orderID := uniqueShippingID()
	otherOrderID := uniqueShippingID()
	cleanup(t, pool, orderID, otherOrderID)

	fixture := seedShippingFixture(t, repo, "it-cust")
	cleanup(t, pool, fixture.ids()...)

	first, err := domain.NewShipment(orderID, "it-user", fixture.Method.ID, fixture.Quote.DestinationSnapshot, 599)
	require.NoError(t, err)
	require.NoError(t, fixture.Quote.Consume())
	require.NoError(t, repo.CreateShipmentForQuote(ctx, first, fixture.Quote))

	second, err := domain.NewShipment(otherOrderID, "it-user", fixture.Method.ID, fixture.Quote.DestinationSnapshot, 599)
	require.NoError(t, err)
	require.ErrorIs(t, repo.CreateShipmentForQuote(ctx, second, fixture.Quote), domain.ErrQuoteConsumed)
}

// TestShippingRepository_MethodAndQuoteRoundTrip proves the reference data the
// shipment depends on survives the round trip through Postgres.
func TestShippingRepository_MethodAndQuoteRoundTrip(t *testing.T) {
	repo, pool := newShippingRepo(t)
	ctx := t.Context()

	fixture := seedShippingFixture(t, repo, "it-cust")
	cleanup(t, pool, fixture.ids()...)

	gotQuote, err := repo.GetQuote(ctx, fixture.Quote.ID)
	require.NoError(t, err)
	assert.Equal(t, fixture.Quote.CustomerID, gotQuote.CustomerID)
	assert.Equal(t, fixture.Quote.ShippingCostCents, gotQuote.ShippingCostCents)
	assert.Equal(t, fixture.Quote.SubtotalCents, gotQuote.SubtotalCents)
	assert.Equal(t, fixture.Quote.FreeThresholdCents, gotQuote.FreeThresholdCents)
	assert.Equal(t, domain.QuoteStatusActive, gotQuote.Status)
	assert.Equal(t, fixture.Quote.DestinationSnapshot, gotQuote.DestinationSnapshot)

	gotMethod, err := repo.GetMethod(ctx, fixture.Method.ID)
	require.NoError(t, err)
	assert.Equal(t, fixture.Method.ID, gotMethod.ID)
	assert.True(t, gotMethod.Active)
}

// TestShippingRepository_RepricedQuoteRoundTrip proves the price a quote is
// re-priced with survives the round trip through Postgres, so the shipment
// created from the quote carries the same cost as the order.
func TestShippingRepository_RepricedQuoteRoundTrip(t *testing.T) {
	repo, pool := newShippingRepo(t)
	ctx := t.Context()

	fixture := seedShippingFixture(t, repo, "it-cust")
	cleanup(t, pool, fixture.ids()...)

	fixture.Quote.ShippingCostCents = 0
	fixture.Quote.SubtotalCents = 6000
	require.NoError(t, repo.UpdateQuote(ctx, fixture.Quote))

	gotQuote, err := repo.GetQuote(ctx, fixture.Quote.ID)
	require.NoError(t, err)
	assert.Equal(t, int64(0), gotQuote.ShippingCostCents)
	assert.Equal(t, int64(6000), gotQuote.SubtotalCents)
	assert.Equal(t, int64(5000), gotQuote.FreeThresholdCents)
}
