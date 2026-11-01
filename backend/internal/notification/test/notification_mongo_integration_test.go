//go:build integration

package test

import (
	"minimart/internal/notification/domain"
	mongorepo "minimart/internal/notification/infrastructure/mongo"
	"sync"
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestRepo(t *testing.T) *mongorepo.NotificationRepository {
	t.Helper()
	client, err := mongo.Connect(options.Client().ApplyURI(testMongoURI))
	require.NoError(t, err)
	t.Cleanup(func() { _ = client.Disconnect(t.Context()) })

	repo := mongorepo.NewNotificationRepository(client.Database("minimart"))
	require.NoError(t, repo.EnsureIndexes(t.Context()))
	return repo
}

func newTestPrefsRepo(t *testing.T) *mongorepo.PreferencesRepository {
	t.Helper()
	client, err := mongo.Connect(options.Client().ApplyURI(testMongoURI))
	require.NoError(t, err)
	t.Cleanup(func() { _ = client.Disconnect(t.Context()) })
	return mongorepo.NewPreferencesRepository(client.Database("minimart"))
}

func newTestDeliveryRepo(t *testing.T) *mongorepo.DeliveryRepository {
	t.Helper()
	client, err := mongo.Connect(options.Client().ApplyURI(testMongoURI))
	require.NoError(t, err)
	t.Cleanup(func() { _ = client.Disconnect(t.Context()) })
	repo := mongorepo.NewDeliveryRepository(client.Database("minimart"))
	require.NoError(t, repo.EnsureIndexes(t.Context()))
	return repo
}

func mustNotification(t *testing.T, id, orderID, userID string) domain.Notification {
	t.Helper()
	n, err := domain.NewOrderConfirmation(id, orderID, userID, 1250)
	require.NoError(t, err)
	return n
}

func TestMongoRepository_SaveGetByID(t *testing.T) {
	ctx := t.Context()
	repo := newTestRepo(t)

	want := mustNotification(t, "it-roundtrip", "it-order-1", "it-user-roundtrip")
	require.NoError(t, repo.Save(ctx, want))

	got, err := repo.GetByIDForUser(ctx, "it-roundtrip", "it-user-roundtrip")
	require.NoError(t, err)
	assert.Equal(t, want.ID, got.ID)
	assert.Equal(t, want.Type, got.Type)
	assert.Equal(t, want.UserID, got.UserID)
	assert.Equal(t, want.OrderID, got.OrderID)
	assert.Equal(t, want.Subject, got.Subject)
	assert.Equal(t, want.Body, got.Body)

	assert.LessOrEqual(t, got.SentAt.Sub(want.SentAt).Abs(), time.Millisecond)
	assert.LessOrEqual(t, got.CreatedAt.Sub(want.CreatedAt).Abs(), time.Millisecond)
}

func TestMongoRepository_SaveIdempotent(t *testing.T) {
	ctx := t.Context()
	repo := newTestRepo(t)

	first := mustNotification(t, "it-idem-1", "it-order-idem", "it-user-idem")
	second := mustNotification(t, "it-idem-2", "it-order-idem", "it-user-idem")

	require.NoError(t, repo.Save(ctx, first))
	require.NoError(t, repo.Save(ctx, second), "a duplicate save must still report success")

	got, err := repo.GetByIDForUser(ctx, "it-idem-1", "it-user-idem")
	require.NoError(t, err)
	assert.Equal(t, first.OrderID, got.OrderID, "the first notification must be kept")

	_, err = repo.GetByIDForUser(ctx, "it-idem-2", "it-user-idem")
	require.ErrorIs(t, err, domain.ErrNotificationNotFound, "the duplicate must not be stored")

	all, err := repo.ListByUserID(ctx, "it-user-idem", 1, 20)
	require.NoError(t, err)
	assert.Len(t, all, 1)
}

func TestMongoRepository_ConcurrentSave(t *testing.T) {
	ctx := t.Context()
	repo := newTestRepo(t)

	const writers = 10

	notifications := make([]domain.Notification, writers)
	for i := range writers {
		notifications[i] = mustNotification(t, "it-race-"+string(rune('a'+i)), "it-order-race", "it-user-race")
	}

	errs := make(chan error, writers)
	var wg sync.WaitGroup
	for i := range writers {
		wg.Go(func() {
			errs <- repo.Save(ctx, notifications[i])
		})
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		require.NoError(t, err)
	}

	all, err := repo.ListByUserID(ctx, "it-user-race", 1, 20)
	require.NoError(t, err)
	assert.Len(t, all, 1, "racing saves of one event must store exactly one notification")
}

func TestMongoRepository_GetByIDNotFound(t *testing.T) {
	repo := newTestRepo(t)

	_, err := repo.GetByIDForUser(t.Context(), "it-missing", "it-user-missing")
	assert.ErrorIs(t, err, domain.ErrNotificationNotFound)
}

func TestMongoRepository_ListByUserID(t *testing.T) {
	ctx := t.Context()
	repo := newTestRepo(t)

	older := mustNotification(t, "it-list-1", "it-order-list-1", "it-user-list")
	newer := mustNotification(t, "it-list-2", "it-order-list-2", "it-user-list")

	newer.CreatedAt = older.CreatedAt.Add(time.Second)
	newer.SentAt = newer.CreatedAt

	require.NoError(t, repo.Save(ctx, older))
	require.NoError(t, repo.Save(ctx, newer))
	require.NoError(t, repo.Save(ctx, mustNotification(t, "it-list-3", "it-order-list-3", "it-user-other")))

	got, err := repo.ListByUserID(ctx, "it-user-list", 1, 20)
	require.NoError(t, err)
	require.Len(t, got, 2, "only this user's notifications are returned")
	assert.Equal(t, "it-list-2", got[0].ID, "newest first")
	assert.Equal(t, "it-list-1", got[1].ID)

	none, err := repo.ListByUserID(ctx, "it-user-nobody", 1, 20)
	require.NoError(t, err)
	assert.Empty(t, none)
}

func TestMongoPreferencesRepository_GetByUserIDNotFound(t *testing.T) {
	repo := newTestPrefsRepo(t)

	_, err := repo.GetByUserID(t.Context(), "it-prefs-missing")
	assert.ErrorIs(t, err, domain.ErrPreferencesNotFound)
}

func TestMongoPreferencesRepository_UpsertRoundTrip(t *testing.T) {
	ctx := t.Context()
	repo := newTestPrefsRepo(t)
	userID := uid(t, "prefs-roundtrip")

	want, err := domain.NewPreferences(userID, false, true, false, true)
	require.NoError(t, err)
	require.NoError(t, repo.Upsert(ctx, want))

	got, err := repo.GetByUserID(ctx, userID)
	require.NoError(t, err)
	assert.Equal(t, want.UserID, got.UserID)
	assert.Equal(t, want.EmailEnabled, got.EmailEnabled)
	assert.Equal(t, want.SMSEnabled, got.SMSEnabled)
	assert.Equal(t, want.PushEnabled, got.PushEnabled)
	assert.Equal(t, want.MarketingEnabled, got.MarketingEnabled)

	assert.LessOrEqual(t, got.UpdatedAt.Sub(want.UpdatedAt).Abs(), time.Millisecond)
}

func TestMongoPreferencesRepository_UpsertReplaces(t *testing.T) {
	ctx := t.Context()
	repo := newTestPrefsRepo(t)
	userID := uid(t, "prefs-replace")

	first, err := domain.NewPreferences(userID, true, true, true, true)
	require.NoError(t, err)
	require.NoError(t, repo.Upsert(ctx, first))

	second, err := domain.NewPreferences(userID, false, false, false, false)
	require.NoError(t, err)
	require.NoError(t, repo.Upsert(ctx, second))

	got, err := repo.GetByUserID(ctx, userID)
	require.NoError(t, err)
	assert.False(t, got.EmailEnabled)
	assert.False(t, got.SMSEnabled)
	assert.False(t, got.PushEnabled)
	assert.False(t, got.MarketingEnabled)
}
