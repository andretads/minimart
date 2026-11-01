package memory

import (
	"minimart/internal/notification/domain"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func mustNotification(t *testing.T, id, orderID, userID string) domain.Notification {
	t.Helper()
	n, err := domain.NewOrderConfirmation(id, orderID, userID, 1250)
	require.NoError(t, err)
	return n
}

func TestRepository(t *testing.T) {
	ctx := t.Context()
	repo := New()

	n := mustNotification(t, "n-1", "o-1", "u-1")

	require.NoError(t, repo.Save(ctx, n))
	got, err := repo.GetByIDForUser(ctx, "n-1", "u-1")
	require.NoError(t, err)
	assert.Equal(t, n, got)

	_, err = repo.GetByIDForUser(ctx, "missing", "u-1")
	require.ErrorIs(t, err, domain.ErrNotificationNotFound)

	require.NoError(t, repo.Save(ctx, mustNotification(t, "n-2", "o-2", "u-1")))
	require.NoError(t, repo.Save(ctx, mustNotification(t, "n-3", "o-3", "u-2")))

	u1, err := repo.ListByUserID(ctx, "u-1", 1, 20)
	require.NoError(t, err)
	assert.Len(t, u1, 2)

	u2, err := repo.ListByUserID(ctx, "u-2", 1, 20)
	require.NoError(t, err)
	require.Len(t, u2, 1)
	assert.Equal(t, "n-3", u2[0].ID)

	none, err := repo.ListByUserID(ctx, "nobody", 1, 20)
	require.NoError(t, err)
	assert.Empty(t, none)
}

func TestRepositorySaveIsIdempotent(t *testing.T) {
	ctx := t.Context()
	repo := New()

	first := mustNotification(t, "n-1", "o-1", "u-1")
	second := mustNotification(t, "n-2", "o-1", "u-1")

	require.NoError(t, repo.Save(ctx, first))
	require.NoError(t, repo.Save(ctx, second), "a duplicate save must still report success")

	got, err := repo.GetByIDForUser(ctx, "n-1", "u-1")
	require.NoError(t, err)
	assert.Equal(t, first, got, "the first notification must be kept untouched")

	_, err = repo.GetByIDForUser(ctx, "n-2", "u-1")
	require.ErrorIs(t, err, domain.ErrNotificationNotFound, "the duplicate must not be stored")

	all, err := repo.ListByUserID(ctx, "u-1", 1, 20)
	require.NoError(t, err)
	assert.Len(t, all, 1)
}

func TestRepositoryListNewestFirst(t *testing.T) {
	ctx := t.Context()
	repo := New()

	older := mustNotification(t, "n-1", "o-1", "u-1")
	newer, err := domain.NewOrderConfirmation("n-2", "o-2", "u-1", 2000)
	require.NoError(t, err)

	newer.CreatedAt = older.CreatedAt.Add(time.Second)
	newer.SentAt = newer.CreatedAt

	require.NoError(t, repo.Save(ctx, older))
	require.NoError(t, repo.Save(ctx, newer))

	got, err := repo.ListByUserID(ctx, "u-1", 1, 20)
	require.NoError(t, err)
	require.Len(t, got, 2)
	assert.Equal(t, "n-2", got[0].ID, "newest first")
	assert.Equal(t, "n-1", got[1].ID)
}

func TestRepositoryConcurrentSave(t *testing.T) {
	ctx := t.Context()
	repo := New()
	n := mustNotification(t, "n-1", "o-1", "u-1")

	errs := make(chan error, 10)
	for range 10 {
		go func() { errs <- repo.Save(ctx, n) }()
	}
	for range 10 {
		require.NoError(t, <-errs)
	}

	all, err := repo.ListByUserID(ctx, "u-1", 1, 20)
	require.NoError(t, err)
	assert.Len(t, all, 1)
}
