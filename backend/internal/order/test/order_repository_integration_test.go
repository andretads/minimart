//go:build integration

// Package test holds the order domain's integration tests.
package test

import (
	"math"
	"minimart/internal/order/domain"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOrderRepository_SaveGetByID(t *testing.T) {
	repo, pool := newOrderRepo(t)
	ctx := t.Context()
	id := uniqueOrderID()
	cleanup(t, pool, id)

	want, err := domain.NewOrder(id, "it-user", "", "", []domain.Item{
		{ProductID: "p1", Name: "Widget", UnitPriceCents: 500, Quantity: 2},
		{ProductID: "p2", Name: "Gadget", UnitPriceCents: 250, Quantity: 1},
	})
	require.NoError(t, err)
	require.NoError(t, repo.Save(ctx, want))

	got, err := repo.GetByID(ctx, id)
	require.NoError(t, err)
	assert.Equal(t, id, got.ID)
	assert.Equal(t, "it-user", got.UserID)
	assert.Equal(t, int64(1250), got.TotalCents)
	require.Len(t, got.Items, 2)
	assert.Equal(t, want.Items, got.Items, "items come back in insertion order with unchanged values")
	assert.False(t, got.CreatedAt.IsZero())
	assert.WithinDuration(t, time.Now().UTC(), got.CreatedAt, time.Minute)
}

func TestOrderRepository_GetByIDNotFound(t *testing.T) {
	repo, _ := newOrderRepo(t)

	_, err := repo.GetByID(t.Context(), uniqueOrderID())
	assert.ErrorIs(t, err, domain.ErrOrderNotFound)
}

// TestOrderRepository_UpdateCancel proves the status/cancelled_at columns survive
// a round trip through the new Update query.
func TestOrderRepository_UpdateCancel(t *testing.T) {
	repo, pool := newOrderRepo(t)
	ctx := t.Context()
	id := uniqueOrderID()
	cleanup(t, pool, id)

	order, err := domain.NewOrder(id, "it-user", "", "", []domain.Item{
		{ProductID: "p1", Name: "Widget", UnitPriceCents: 500, Quantity: 1},
	})
	require.NoError(t, err)
	require.NoError(t, repo.Save(ctx, order))

	// The database default stamps a freshly saved order as placed.
	stored, err := repo.GetByID(ctx, id)
	require.NoError(t, err)
	assert.Equal(t, domain.StatusPlaced, stored.Status)
	assert.Nil(t, stored.CancelledAt)

	cancelled, err := stored.Cancel()
	require.NoError(t, err)
	require.NoError(t, repo.Update(ctx, cancelled, domain.StatusPlaced))

	got, err := repo.GetByID(ctx, id)
	require.NoError(t, err)
	assert.Equal(t, domain.StatusCancelled, got.Status)
	require.NotNil(t, got.CancelledAt)
	assert.WithinDuration(t, *cancelled.CancelledAt, *got.CancelledAt, time.Second)
}

func TestOrderRepository_UpdateNotFound(t *testing.T) {
	repo, _ := newOrderRepo(t)

	order, err := domain.NewOrder(uniqueOrderID(), "it-user", "", "", []domain.Item{
		{ProductID: "p1", Name: "Widget", UnitPriceCents: 500, Quantity: 1},
	})
	require.NoError(t, err)

	assert.ErrorIs(t, repo.Update(t.Context(), order, domain.StatusPlaced), domain.ErrOrderNotFound)
}

// TestOrderRepository_UpdateStaleStatusReportsStateChanged proves the compare-and-
// set: writing with an expected status that no longer matches changes nothing and
// reports ErrOrderStateChanged, so a concurrent transition cannot be clobbered.
func TestOrderRepository_UpdateStaleStatusReportsStateChanged(t *testing.T) {
	repo, pool := newOrderRepo(t)
	ctx := t.Context()
	id := uniqueOrderID()
	cleanup(t, pool, id)

	order, err := domain.NewOrder(id, "it-user", "", "", []domain.Item{
		{ProductID: "p1", Name: "Widget", UnitPriceCents: 500, Quantity: 1},
	})
	require.NoError(t, err)
	require.NoError(t, repo.Save(ctx, order))

	// First transition wins: placed -> paid.
	require.NoError(t, repo.Update(ctx, order.MarkPaid(), domain.StatusPlaced))

	// A second writer still holding the old (placed) status loses the race.
	stale := order.MarkPaymentFailed()
	err = repo.Update(ctx, stale, domain.StatusPlaced)
	require.ErrorIs(t, err, domain.ErrOrderStateChanged)

	got, err := repo.GetByID(ctx, id)
	require.NoError(t, err)
	assert.Equal(t, domain.StatusPaid, got.Status, "the losing write must not change the order")
}

// TestOrderRepository_SaveRejectsInt32QuantityAndRollsBack proves two things at
// once: the quantity range guard fires, and the deferred Rollback undoes the order
// row that was already inserted earlier in the same transaction. No outbox row
// may be left behind either.
func TestOrderRepository_SaveRejectsInt32QuantityAndRollsBack(t *testing.T) {
	repo, pool := newOrderRepo(t)
	ctx := t.Context()
	id := uniqueOrderID()
	cleanup(t, pool, id)

	order := domain.Order{
		ID:     id,
		UserID: "it-user",
		Items: []domain.Item{
			{ProductID: "p1", Name: "Widget", UnitPriceCents: 500, Quantity: math.MaxInt32 + 1},
		},
	}

	err := repo.Save(ctx, order)
	require.ErrorContains(t, err, "int32")

	_, err = repo.GetByID(ctx, id)
	require.ErrorIs(t, err, domain.ErrOrderNotFound, "the partially written order must be rolled back")
	assert.Zero(t, countOutbox(ctx, t, pool, id), "no outbox event may survive a rolled-back save")
}

func TestOrderRepository_ListByUserID(t *testing.T) {
	repo, pool := newOrderRepo(t)
	ctx := t.Context()
	userID := uniqueOrderID()
	otherUserID := userID + "-other"

	saveOrder := func(t *testing.T, id string, user string) {
		t.Helper()
		cleanup(t, pool, id)
		o, err := domain.NewOrder(id, user, "", "", []domain.Item{
			{ProductID: "p1", Name: "Widget", UnitPriceCents: 500, Quantity: 1},
		})
		require.NoError(t, err)
		require.NoError(t, repo.Save(ctx, o))
		time.Sleep(2 * time.Millisecond)
	}
	id1 := userID + "-1"
	id2 := userID + "-2"
	id3 := userID + "-3"
	otherID := otherUserID + "-1"
	saveOrder(t, id1, userID)
	saveOrder(t, id2, userID)
	saveOrder(t, id3, userID)
	saveOrder(t, otherID, otherUserID)

	t.Run("lists only the user's orders newest first with items", func(t *testing.T) {
		got, err := repo.ListByUserID(ctx, userID, 20, 0)
		require.NoError(t, err)
		require.Len(t, got, 3)
		assert.Equal(t, id3, got[0].ID)
		assert.Equal(t, id2, got[1].ID)
		assert.Equal(t, id1, got[2].ID)
		for _, o := range got {
			require.Len(t, o.Items, 1)
			assert.Equal(t, "p1", o.Items[0].ProductID)
		}
	})

	t.Run("applies limit and offset", func(t *testing.T) {
		got, err := repo.ListByUserID(ctx, userID, 1, 1)
		require.NoError(t, err)
		require.Len(t, got, 1)
		assert.Equal(t, id2, got[0].ID)
	})

	t.Run("counts the user's orders", func(t *testing.T) {
		count, err := repo.CountByUserID(ctx, userID)
		require.NoError(t, err)
		assert.Equal(t, 3, count)

		count, err = repo.CountByUserID(ctx, uniqueOrderID())
		require.NoError(t, err)
		assert.Zero(t, count)
	})
}
