package memory

import (
	"minimart/internal/order/domain"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRepository(t *testing.T) {
	ctx := t.Context()
	repo := New()

	want, err := domain.NewOrder("o-1", "u-1", "cust-1", "k1", []domain.Item{
		{ProductID: "p-1", Name: "Widget", UnitPriceCents: 500, Quantity: 2},
	})
	require.NoError(t, err)

	// Save, then read the same order back.
	require.NoError(t, repo.Save(ctx, want))
	got, err := repo.GetByID(ctx, "o-1")
	require.NoError(t, err)
	assert.Equal(t, want, got)

	// A missing id returns the domain not-found sentinel.
	_, err = repo.GetByID(ctx, "missing")
	assert.ErrorIs(t, err, domain.ErrOrderNotFound)
}

func TestRepositoryUpdate(t *testing.T) {
	ctx := t.Context()
	repo := New()

	o, err := domain.NewOrder("o-1", "u-1", "cust-1", "k1", []domain.Item{
		{ProductID: "p-1", Name: "Widget", UnitPriceCents: 500, Quantity: 1},
	})
	require.NoError(t, err)
	require.NoError(t, repo.Save(ctx, o))

	cancelled, err := o.Cancel()
	require.NoError(t, err)
	require.NoError(t, repo.Update(ctx, cancelled, domain.StatusPlaced))

	got, err := repo.GetByID(ctx, "o-1")
	require.NoError(t, err)
	assert.Equal(t, domain.StatusCancelled, got.Status)
	require.NotNil(t, got.CancelledAt)

	t.Run("missing id returns not found", func(t *testing.T) {
		absent, err := domain.NewOrder("missing", "u-1", "cust-1", "k2", []domain.Item{
			{ProductID: "p-1", Name: "Widget", UnitPriceCents: 500, Quantity: 1},
		})
		require.NoError(t, err)
		assert.ErrorIs(t, repo.Update(ctx, absent, domain.StatusPlaced), domain.ErrOrderNotFound)
	})

	t.Run("stale expected status returns state changed", func(t *testing.T) {
		paid := cancelled.MarkPaid()
		assert.ErrorIs(t, repo.Update(ctx, paid, domain.StatusPlaced), domain.ErrOrderStateChanged)
	})
}

func TestRepositoryListByUserID(t *testing.T) {
	ctx := t.Context()
	repo := New()
	now := time.Now().UTC()

	save := func(t *testing.T, id, userID string, createdAt time.Time) {
		t.Helper()
		o, err := domain.NewOrder(id, userID, "", "", []domain.Item{
			{ProductID: "p-1", Name: "Widget", UnitPriceCents: 500, Quantity: 1},
		})
		require.NoError(t, err)
		o.CreatedAt = createdAt
		require.NoError(t, repo.Save(ctx, o))
	}
	save(t, "o-1", "u-1", now.Add(-3*time.Hour))
	save(t, "o-2", "u-1", now.Add(-2*time.Hour))
	save(t, "o-3", "u-1", now.Add(-1*time.Hour))
	save(t, "o-4", "u-2", now)

	t.Run("filters by user and orders newest first", func(t *testing.T) {
		got, err := repo.ListByUserID(ctx, "u-1", 20, 0)
		require.NoError(t, err)
		require.Len(t, got, 3)
		assert.Equal(t, "o-3", got[0].ID)
		assert.Equal(t, "o-2", got[1].ID)
		assert.Equal(t, "o-1", got[2].ID)
	})

	t.Run("applies limit and offset", func(t *testing.T) {
		got, err := repo.ListByUserID(ctx, "u-1", 1, 1)
		require.NoError(t, err)
		require.Len(t, got, 1)
		assert.Equal(t, "o-2", got[0].ID)
	})

	t.Run("offset beyond the end returns empty", func(t *testing.T) {
		got, err := repo.ListByUserID(ctx, "u-1", 20, 10)
		require.NoError(t, err)
		assert.Empty(t, got)
	})

	t.Run("counts only the requested user's orders", func(t *testing.T) {
		count, err := repo.CountByUserID(ctx, "u-1")
		require.NoError(t, err)
		assert.Equal(t, 3, count)

		count, err = repo.CountByUserID(ctx, "missing")
		require.NoError(t, err)
		assert.Zero(t, count)
	})
}

func TestRepositoryListPlacedBefore(t *testing.T) {
	ctx := t.Context()
	repo := New()
	now := time.Now().UTC()

	save := func(t *testing.T, id, status string, createdAt time.Time) {
		t.Helper()
		o, err := domain.NewOrder(id, "u-1", "", "", []domain.Item{
			{ProductID: "p-1", Name: "Widget", UnitPriceCents: 500, Quantity: 1},
		})
		require.NoError(t, err)
		o.CreatedAt = createdAt
		o.Status = domain.Status(status)
		require.NoError(t, repo.Save(ctx, o))
	}
	save(t, "placed-old", "placed", now.Add(-10*time.Minute))
	save(t, "placed-new", "placed", now.Add(-1*time.Minute))
	save(t, "paid-old", "paid", now.Add(-10*time.Minute))
	save(t, "cancelled-old", "cancelled", now.Add(-10*time.Minute))

	t.Run("returns only placed orders at or before the cutoff, oldest first", func(t *testing.T) {
		got, err := repo.ListPlacedBefore(ctx, now.Add(-5*time.Minute), 10)
		require.NoError(t, err)
		require.Len(t, got, 1)
		assert.Equal(t, "placed-old", got[0].ID)
	})

	t.Run("applies the limit", func(t *testing.T) {
		save(t, "placed-mid", "placed", now.Add(-7*time.Minute))
		got, err := repo.ListPlacedBefore(ctx, now.Add(-5*time.Minute), 1)
		require.NoError(t, err)
		require.Len(t, got, 1)
		assert.Equal(t, "placed-old", got[0].ID)
	})

	t.Run("returns an empty slice when nothing matches", func(t *testing.T) {
		got, err := repo.ListPlacedBefore(ctx, now.Add(-1*time.Hour), 10)
		require.NoError(t, err)
		assert.Empty(t, got)
	})
}
