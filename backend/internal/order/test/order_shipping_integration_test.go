//go:build integration

package test

import (
	"encoding/json"
	"minimart/internal/order/application"
	"minimart/internal/order/domain"
	"minimart/pkg/kafka"
	"minimart/pkg/message"
	"minimart/pkg/metrics"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// shippingDestination is the address snapshot used by the shipping tests.
func shippingDestination() domain.DestinationSnapshot {
	return domain.DestinationSnapshot{
		Label:    "Home",
		FullName: "Ada Lovelace",
		Street:   "1 Main St",
		City:     "Testville",
		State:    "TS",
		Zip:      "12345",
		Country:  "BR",
		Phone:    "555-0100",
	}
}

// orderWithShipping builds a placed order that carries a shipping snapshot,
// as the cart checkout places it.
func orderWithShipping(t *testing.T, id string) domain.Order {
	t.Helper()
	order, err := domain.NewOrder(id, "it-user", "", "", []domain.Item{
		{ProductID: "p1", Name: "Widget", UnitPriceCents: 500, Quantity: 1},
	})
	require.NoError(t, err)
	order.ShippingQuoteID = "quote-1"
	order.ShippingMethodID = "method-1"
	order.ShippingMethodName = "Standard"
	order.ShippingCostCents = 250
	order.DestinationSnapshot = shippingDestination()
	order.TotalCents += order.ShippingCostCents
	return order
}

// TestOrderRepository_ShippingSnapshotRoundTrip proves the shipping snapshot
// taken at placement survives a round trip through Postgres: the order the
// payment consumer reloads still carries it, which is what makes the
// order-paid to shipment handoff possible.
func TestOrderRepository_ShippingSnapshotRoundTrip(t *testing.T) {
	repo, pool := newOrderRepo(t)
	ctx := t.Context()
	id := uniqueOrderID()
	cleanup(t, pool, id)

	want := orderWithShipping(t, id)
	require.NoError(t, repo.Save(ctx, want))

	got, err := repo.GetByID(ctx, id)
	require.NoError(t, err)
	assert.Equal(t, "quote-1", got.ShippingQuoteID)
	assert.Equal(t, "method-1", got.ShippingMethodID)
	assert.Equal(t, "Standard", got.ShippingMethodName)
	assert.Equal(t, int64(250), got.ShippingCostCents)
	assert.Equal(t, shippingDestination(), got.DestinationSnapshot)
	assert.Equal(t, int64(750), got.TotalCents)
}

// TestOrderRepository_ShippingSnapshotIsListed proves the list path returns the
// snapshot too, so the order list can show the shipping cost.
func TestOrderRepository_ShippingSnapshotIsListed(t *testing.T) {
	repo, pool := newOrderRepo(t)
	ctx := t.Context()
	id := uniqueOrderID()
	cleanup(t, pool, id)

	require.NoError(t, repo.Save(ctx, orderWithShipping(t, id)))

	got, err := repo.ListByUserID(ctx, "it-user", 20, 0)
	require.NoError(t, err)
	require.Len(t, got, 1)
	assert.Equal(t, "quote-1", got[0].ShippingQuoteID)
	assert.Equal(t, "method-1", got[0].ShippingMethodID)
	assert.Equal(t, int64(250), got[0].ShippingCostCents)
	assert.Equal(t, shippingDestination(), got[0].DestinationSnapshot)
}

// TestApplyPaymentOutcome_WritesOrderPaidEvent proves the handoff end to end in
// the database: a payment outcome on an order that carries the shipping
// snapshot writes an order-paid outbox row carrying that snapshot, which is
// exactly what the shipping domain consumes to create the shipment.
func TestApplyPaymentOutcome_WritesOrderPaidEvent(t *testing.T) {
	repo, pool := newOrderRepo(t)
	ctx := t.Context()
	id := uniqueOrderID()
	cleanup(t, pool, id)

	require.NoError(t, repo.Save(ctx, orderWithShipping(t, id)))

	svc := application.NewService(repo, nil, nil, metrics.Noop{}, 0)
	require.NoError(t, svc.ApplyPaymentOutcome(ctx, id, application.PaymentOutcomeSucceeded, ""))

	got, err := repo.GetByID(ctx, id)
	require.NoError(t, err)
	assert.Equal(t, domain.StatusPaid, got.Status)
	assert.Equal(t, "quote-1", got.ShippingQuoteID, "the snapshot survives the payment write-back")
	assert.Equal(t, shippingDestination(), got.DestinationSnapshot)

	var event message.OrderPaid
	require.NoError(t, json.Unmarshal(outboxBody(ctx, t, pool, id, kafka.OrderPaidTopic.String()), &event))
	assert.Equal(t, id, event.OrderID)
	assert.Equal(t, "it-user", event.UserID)
	assert.Equal(t, int64(750), event.TotalCents)
	assert.Equal(t, "quote-1", event.ShippingQuoteID)
	assert.Equal(t, "method-1", event.ShippingMethodID)
	assert.Equal(t, "Ada Lovelace", event.DestinationSnapshot.FullName)
	assert.Equal(t, "1 Main St", event.DestinationSnapshot.Street)
	assert.Equal(t, "555-0100", event.DestinationSnapshot.Phone)
}

// TestApplyPaymentOutcome_WithoutShippingWritesNoOrderPaidEvent proves an order
// placed without shipping has nothing to hand over, so no order-paid event is
// written and no shipment is created for it.
func TestApplyPaymentOutcome_WithoutShippingWritesNoOrderPaidEvent(t *testing.T) {
	repo, pool := newOrderRepo(t)
	ctx := t.Context()
	id := uniqueOrderID()
	cleanup(t, pool, id)

	order, err := domain.NewOrder(id, "it-user", "", "", []domain.Item{
		{ProductID: "p1", Name: "Widget", UnitPriceCents: 500, Quantity: 1},
	})
	require.NoError(t, err)
	require.NoError(t, repo.Save(ctx, order))

	svc := application.NewService(repo, nil, nil, metrics.Noop{}, 0)
	require.NoError(t, svc.ApplyPaymentOutcome(ctx, id, application.PaymentOutcomeSucceeded, ""))

	got, err := repo.GetByID(ctx, id)
	require.NoError(t, err)
	assert.Equal(t, domain.StatusPaid, got.Status)
	assert.Empty(t, got.ShippingQuoteID)

	topics := outboxTopics(ctx, t, pool, id)
	assert.NotContains(t, topics, kafka.OrderPaidTopic.String(), "there is no shipment to hand over")
}
