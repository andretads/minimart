//go:build integration

// Package test holds the order domain's integration tests.
package test

import (
	"encoding/json"
	"minimart/internal/order/domain"
	"minimart/pkg/kafka"
	"minimart/pkg/message"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestOrderRepository_SaveWritesOutboxEvent proves the order and its OrderPlaced
// event commit together: after a successful save, a pending outbox row exists with
// the event's topic, key, and serialized payload.
func TestOrderRepository_SaveWritesOutboxEvent(t *testing.T) {
	repo, pool := newOrderRepo(t)
	ctx := t.Context()
	id := uniqueOrderID()
	cleanup(t, pool, id)

	order, err := domain.NewOrder(id, "it-user", "", "", []domain.Item{
		{ProductID: "p1", Name: "Widget", UnitPriceCents: 500, Quantity: 2},
	})
	require.NoError(t, err)
	require.NoError(t, repo.Save(ctx, order))

	var (
		topic string
		key   string
		body  []byte
		state int
	)
	err = pool.QueryRow(ctx,
		"SELECT topic, message_key, body, state FROM outbox WHERE message_key = $1", id,
	).Scan(&topic, &key, &body, &state)
	require.NoError(t, err)

	assert.Equal(t, kafka.OrderPlacedTopic.String(), topic)
	assert.Equal(t, id, key)
	assert.Equal(t, 0, state, "a freshly written event is pending")
	var payload message.OrderPlaced
	require.NoError(t, json.Unmarshal(body, &payload))
	assert.Equal(t, id, payload.OrderID)
	assert.Equal(t, int64(1000), payload.TotalCents)
	assert.Equal(t, 1, payload.ItemCount)
}

// TestOrderRepository_UpdateCancelWritesOutboxEvent proves a cancellation writes
// both the OrderCancelled event and the OrderStockRelease event to the outbox
// alongside the status change, so the catalog can free the held stock.
func TestOrderRepository_UpdateCancelWritesOutboxEvent(t *testing.T) {
	repo, pool := newOrderRepo(t)
	ctx := t.Context()
	id := uniqueOrderID()
	cleanup(t, pool, id)

	order, err := domain.NewOrder(id, "it-user", "", "", []domain.Item{
		{ProductID: "p1", Name: "Widget", UnitPriceCents: 500, Quantity: 1},
	})
	require.NoError(t, err)
	require.NoError(t, repo.Save(ctx, order))

	// Clear the placed event so only the cancellation events remain to assert on.
	_, err = pool.Exec(ctx, "DELETE FROM outbox WHERE message_key = $1", id)
	require.NoError(t, err)

	stored, err := repo.GetByID(ctx, id)
	require.NoError(t, err)
	cancelled, err := stored.Cancel()
	require.NoError(t, err)
	require.NoError(t, repo.Update(ctx, cancelled, domain.StatusPlaced))

	assert.ElementsMatch(t, []string{
		kafka.OrderCancelledTopic.String(),
		kafka.OrderStockReleaseTopic.String(),
	}, outboxTopics(ctx, t, pool, id))
}

// TestOrderRepository_UpdatePaymentFailedWritesStockReleaseEvent proves a failed
// payment releases the order's stock reservation through the outbox.
func TestOrderRepository_UpdatePaymentFailedWritesStockReleaseEvent(t *testing.T) {
	repo, pool := newOrderRepo(t)
	ctx := t.Context()
	id := uniqueOrderID()
	cleanup(t, pool, id)

	order, err := domain.NewOrder(id, "it-user", "", "", []domain.Item{
		{ProductID: "p1", Name: "Widget", UnitPriceCents: 500, Quantity: 1},
	})
	require.NoError(t, err)
	require.NoError(t, repo.Save(ctx, order))
	_, err = pool.Exec(ctx, "DELETE FROM outbox WHERE message_key = $1", id)
	require.NoError(t, err)

	stored, err := repo.GetByID(ctx, id)
	require.NoError(t, err)
	require.NoError(t, repo.Update(ctx, stored.MarkPaymentFailed(), domain.StatusPlaced))

	assert.ElementsMatch(t, []string{kafka.OrderStockReleaseTopic.String()}, outboxTopics(ctx, t, pool, id))
}

// TestOrderRepository_UpdatePaymentUnknownHoldsStock proves that a payment_unknown
// outcome does NOT release stock - the reservation is held.
func TestOrderRepository_UpdatePaymentUnknownHoldsStock(t *testing.T) {
	repo, pool := newOrderRepo(t)
	ctx := t.Context()
	id := uniqueOrderID()
	cleanup(t, pool, id)

	order, err := domain.NewOrder(id, "it-user", "", "", []domain.Item{
		{ProductID: "p1", Name: "Widget", UnitPriceCents: 500, Quantity: 1},
	})
	require.NoError(t, err)
	require.NoError(t, repo.Save(ctx, order))
	_, err = pool.Exec(ctx, "DELETE FROM outbox WHERE message_key = $1", id)
	require.NoError(t, err)

	stored, err := repo.GetByID(ctx, id)
	require.NoError(t, err)
	require.NoError(t, repo.Update(ctx, stored.MarkPaymentUnknown(), domain.StatusPlaced))

	// No stock release event should be written for payment_unknown
	assert.ElementsMatch(t, []string{}, outboxTopics(ctx, t, pool, id))
}

// TestOrderRepository_UpdatePaidNoStockRelease proves that a successful payment
// does NOT release stock - the order is paid and stock remains reserved.
func TestOrderRepository_UpdatePaidNoStockRelease(t *testing.T) {
	repo, pool := newOrderRepo(t)
	ctx := t.Context()
	id := uniqueOrderID()
	cleanup(t, pool, id)

	order, err := domain.NewOrder(id, "it-user", "", "", []domain.Item{
		{ProductID: "p1", Name: "Widget", UnitPriceCents: 500, Quantity: 1},
	})
	require.NoError(t, err)
	require.NoError(t, repo.Save(ctx, order))
	_, err = pool.Exec(ctx, "DELETE FROM outbox WHERE message_key = $1", id)
	require.NoError(t, err)

	stored, err := repo.GetByID(ctx, id)
	require.NoError(t, err)
	require.NoError(t, repo.Update(ctx, stored.MarkPaid(), domain.StatusPlaced))

	// No stock release event should be written for paid
	assert.ElementsMatch(t, []string{}, outboxTopics(ctx, t, pool, id))
}
