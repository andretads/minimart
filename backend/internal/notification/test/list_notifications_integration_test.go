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

func TestListNotificationsIntegration(t *testing.T) {
	ctx := t.Context()
	repo := newTestRepo(t)
	deliveryRepo := newTestDeliveryRepo(t)
	prefsRepo := newTestPrefsRepo(t)
	svc := application.NewService(repo, deliveryRepo, prefsRepo, newFakeEmailSender(), nil, nil, newFakeRecipientResolver(), nil, metrics.Noop{}, application.RetryConfig{})

	t.Run("Missing user_id is rejected", func(t *testing.T) {
		_, _, err := svc.ListNotifications(ctx, "", 1, 20)
		require.ErrorIs(t, err, domain.ErrEmptyUserID)
	})

	t.Run("User with no notifications gets an empty page and a zero total", func(t *testing.T) {
		got, total, err := svc.ListNotifications(ctx, uid(t, "nouser"), 1, 20)
		require.NoError(t, err)
		assert.Empty(t, got)
		assert.Equal(t, 0, total)
	})

	t.Run("Only the user's notifications are returned, newest first", func(t *testing.T) {
		userID := uid(t, "u1")
		otherUser := uid(t, "u2")
		require.NoError(t, repo.Save(ctx, mustNotification(t, uid(t, "n1"), uid(t, "o1"), userID)))
		require.NoError(t, repo.Save(ctx, mustNotification(t, uid(t, "n2"), uid(t, "o2"), userID)))
		require.NoError(t, repo.Save(ctx, mustNotification(t, uid(t, "n3"), uid(t, "o3"), otherUser)))

		got, total, err := svc.ListNotifications(ctx, userID, 1, 20)
		require.NoError(t, err)
		assert.Len(t, got, 2)
		assert.Equal(t, 2, total)
	})

	t.Run("Pagination returns the requested page", func(t *testing.T) {
		userID := uid(t, "u1p")
		require.NoError(t, repo.Save(ctx, mustNotification(t, uid(t, "n1"), uid(t, "o1"), userID)))
		require.NoError(t, repo.Save(ctx, mustNotification(t, uid(t, "n2"), uid(t, "o2"), userID)))
		require.NoError(t, repo.Save(ctx, mustNotification(t, uid(t, "n3"), uid(t, "o3"), userID)))

		got, _, err := svc.ListNotifications(ctx, userID, 2, 2)
		require.NoError(t, err)
		assert.Len(t, got, 1)
	})
}
