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

func TestMarkAllNotificationsReadIntegration(t *testing.T) {
	ctx := t.Context()
	repo := newTestRepo(t)
	deliveryRepo := newTestDeliveryRepo(t)
	prefsRepo := newTestPrefsRepo(t)
	svc := application.NewService(repo, deliveryRepo, prefsRepo, newFakeEmailSender(), nil, nil, newFakeRecipientResolver(), nil, metrics.Noop{}, application.RetryConfig{})

	t.Run("Empty user_id is rejected", func(t *testing.T) {
		err := svc.MarkAllNotificationsRead(ctx, "")
		require.ErrorIs(t, err, domain.ErrEmptyUserID)
	})

	t.Run("A user with no notifications gets a no-op", func(t *testing.T) {
		err := svc.MarkAllNotificationsRead(ctx, uid(t, "none"))
		require.NoError(t, err)
	})

	t.Run("All of the user's notifications are marked as read", func(t *testing.T) {
		userID := uid(t, "u1b")
		n1 := mustNotification(t, uid(t, "n1"), uid(t, "o1"), userID)
		n2 := mustNotification(t, uid(t, "n2"), uid(t, "o2"), userID)
		require.NoError(t, repo.Save(ctx, n1))
		require.NoError(t, repo.Save(ctx, n2))
		err := svc.MarkAllNotificationsRead(ctx, userID)
		require.NoError(t, err)
		got1, _ := repo.GetByIDForUser(ctx, n1.ID, userID)
		got2, _ := repo.GetByIDForUser(ctx, n2.ID, userID)
		assert.NotNil(t, got1.ReadAt)
		assert.NotNil(t, got2.ReadAt)
	})

	t.Run("Another user's notifications are left untouched", func(t *testing.T) {
		userID := uid(t, "u1c")
		other := uid(t, "u2c")
		n1 := mustNotification(t, uid(t, "n1"), uid(t, "o1"), userID)
		n2 := mustNotification(t, uid(t, "n2"), uid(t, "o2"), other)
		require.NoError(t, repo.Save(ctx, n1))
		require.NoError(t, repo.Save(ctx, n2))
		err := svc.MarkAllNotificationsRead(ctx, userID)
		require.NoError(t, err)
		got2, _ := repo.GetByIDForUser(ctx, n2.ID, other)
		assert.Nil(t, got2.ReadAt)
	})
}
