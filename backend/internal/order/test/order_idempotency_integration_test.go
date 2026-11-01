//go:build integration

// Package test holds the order domain's integration tests.
package test

import (
	"minimart/internal/order/domain"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestOrderRepository_IdempotencyKey proves the key round-trips and that the
// partial unique index rejects a second order with the same non-empty key.
func TestOrderRepository_IdempotencyKey(t *testing.T) {
	repo, pool := newOrderRepo(t)
	ctx := t.Context()
	id := uniqueOrderID()
	key := "it-idem-" + id
	cleanup(t, pool, id)

	order, err := domain.NewOrder(id, "it-user", "", key, []domain.Item{
		{ProductID: "p1", Name: "Widget", UnitPriceCents: 500, Quantity: 1},
	})
	require.NoError(t, err)
	require.NoError(t, repo.Save(ctx, order))

	got, err := repo.GetByIdempotencyKey(ctx, key)
	require.NoError(t, err)
	assert.Equal(t, id, got.ID)
	assert.Equal(t, key, got.IdempotencyKey)

	_, err = repo.GetByIdempotencyKey(ctx, "it-idem-missing-"+id)
	require.ErrorIs(t, err, domain.ErrOrderNotFound)

	// A second order reusing the key violates the partial unique index, which the
	// repository reports as ErrIdempotencyConflict so the caller can replay.
	dup, err := domain.NewOrder(uniqueOrderID(), "it-user", "", key, []domain.Item{
		{ProductID: "p1", Name: "Widget", UnitPriceCents: 500, Quantity: 1},
	})
	require.NoError(t, err)
	cleanup(t, pool, dup.ID)
	require.ErrorIs(t, repo.Save(ctx, dup), domain.ErrIdempotencyConflict)

	// An empty key is outside the partial index and never conflicts.
	first, err := domain.NewOrder(uniqueOrderID(), "it-user", "", "", []domain.Item{
		{ProductID: "p1", Name: "Widget", UnitPriceCents: 500, Quantity: 1},
	})
	require.NoError(t, err)
	cleanup(t, pool, first.ID)
	require.NoError(t, repo.Save(ctx, first))

	second, err := domain.NewOrder(uniqueOrderID(), "it-user", "", "", []domain.Item{
		{ProductID: "p1", Name: "Widget", UnitPriceCents: 500, Quantity: 1},
	})
	require.NoError(t, err)
	cleanup(t, pool, second.ID)
	require.NoError(t, repo.Save(ctx, second))
}
