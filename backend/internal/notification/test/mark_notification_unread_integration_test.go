//go:build integration

package test

import (
	"minimart/internal/notification/application"
	"minimart/internal/notification/domain"
	"minimart/pkg/metrics"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMarkNotificationUnreadIntegration(t *testing.T) {
	ctx := t.Context()
	repo := newTestRepo(t)
	deliveryRepo := newTestDeliveryRepo(t)
	prefsRepo := newTestPrefsRepo(t)
	svc := application.NewService(repo, deliveryRepo, prefsRepo, newFakeEmailSender(), nil, nil, newFakeRecipientResolver(), nil, metrics.Noop{}, application.RetryConfig{})

	t.Run("Empty notification id is rejected", func(t *testing.T) {
		err := svc.MarkNotificationUnread(ctx, "", "u1")
		require.ErrorIs(t, err, domain.ErrEmptyID)
	})

	t.Run("Empty user_id is rejected", func(t *testing.T) {
		err := svc.MarkNotificationUnread(ctx, "n1", "")
		require.ErrorIs(t, err, domain.ErrEmptyUserID)
	})

	t.Run("Notification does not exist", func(t *testing.T) {
		err := svc.MarkNotificationUnread(ctx, uid(t, "missing"), "u1")
		require.ErrorIs(t, err, domain.ErrNotificationNotFound)
	})

	t.Run("Another user's notification is not found", func(t *testing.T) {
		userID := uid(t, "u1")
		n := mustNotification(t, uid(t, "n1"), uid(t, "o1"), userID)
		now := time.Now().UTC()
		n.ReadAt = &now
		require.NoError(t, repo.Save(ctx, n))
		err := svc.MarkNotificationUnread(ctx, n.ID, uid(t, "u2"))
		require.ErrorIs(t, err, domain.ErrNotificationNotFound)
	})

	t.Run("The notification is marked as unread", func(t *testing.T) {
		userID := uid(t, "u1b")
		n := mustNotification(t, uid(t, "n1"), uid(t, "o1"), userID)
		now := time.Now().UTC()
		n.ReadAt = &now
		require.NoError(t, repo.Save(ctx, n))
		err := svc.MarkNotificationUnread(ctx, n.ID, userID)
		require.NoError(t, err)
		got, err := repo.GetByIDForUser(ctx, n.ID, userID)
		require.NoError(t, err)
		assert.Nil(t, got.ReadAt)
	})
}
