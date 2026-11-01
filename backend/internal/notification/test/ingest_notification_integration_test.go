//go:build integration

package test

import (
	"minimart/internal/notification/application"
	"minimart/internal/notification/domain"
	"minimart/pkg/metrics"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIngestNotificationIntegration(t *testing.T) {
	ctx := t.Context()
	repo := newTestRepo(t)
	deliveries := newTestDeliveryRepo(t)
	prefs := newTestPrefsRepo(t)
	resolver := newFakeRecipientResolver()
	svc := application.NewService(repo, deliveries, prefs, newFakeEmailSender(), nil, nil, resolver, nil, metrics.Noop{}, application.RetryConfig{})

	t.Run("A duplicate dedupe key reuses the stored notification and its deliveries", func(t *testing.T) {
		userID := uid(t, "it-user-dedupe")
		resolver.SetEmail(userID, "dedupe@example.com")
		p, _ := domain.NewPreferences(userID, true, false, false, false)
		require.NoError(t, prefs.Upsert(ctx, p))

		first := mustNotification(t, uid(t, "it-dedupe-1"), uid(t, "it-order-dedupe"), userID)
		persisted, created, err := svc.IngestNotification(ctx, first, nil)
		require.NoError(t, err)
		require.Len(t, created, 1)

		duplicate := mustNotification(t, uid(t, "it-dedupe-2"), first.OrderID, userID)
		got, reused, err := svc.IngestNotification(ctx, duplicate, nil)
		require.NoError(t, err)

		assert.Equal(t, persisted.ID, got.ID)
		assert.Len(t, reused, 1)

		all, err := repo.ListByUserID(ctx, userID, 1, 20)
		require.NoError(t, err)
		assert.Len(t, all, 1)

		orphans, err := deliveries.ListByNotification(ctx, duplicate.ID)
		require.NoError(t, err)
		assert.Empty(t, orphans)

		stored, err := deliveries.ListByNotification(ctx, persisted.ID)
		require.NoError(t, err)
		assert.Len(t, stored, 1)

		_, err = repo.GetByID(ctx, duplicate.ID)
		assert.ErrorIs(t, err, domain.ErrNotificationNotFound)
	})
}
