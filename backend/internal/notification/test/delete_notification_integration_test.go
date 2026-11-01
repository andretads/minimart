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

func TestDeleteNotificationIntegration(t *testing.T) {
	ctx := t.Context()
	repo := newTestRepo(t)
	deliveryRepo := newTestDeliveryRepo(t)
	prefsRepo := newTestPrefsRepo(t)
	svc := application.NewService(repo, deliveryRepo, prefsRepo, newFakeEmailSender(), nil, nil, newFakeRecipientResolver(), nil, metrics.Noop{}, application.RetryConfig{})

	t.Run("Empty notification id is rejected", func(t *testing.T) {
		err := svc.DeleteNotification(ctx, "", "u1")
		require.ErrorIs(t, err, domain.ErrEmptyID)
	})

	t.Run("Empty user_id is rejected", func(t *testing.T) {
		err := svc.DeleteNotification(ctx, "n1", "")
		require.ErrorIs(t, err, domain.ErrEmptyUserID)
	})

	t.Run("Notification does not exist", func(t *testing.T) {
		err := svc.DeleteNotification(ctx, uid(t, "missing"), "u1")
		require.ErrorIs(t, err, domain.ErrNotificationNotFound)
	})

	t.Run("Another user's notification is not found", func(t *testing.T) {
		userID := uid(t, "u1")
		n := mustNotification(t, uid(t, "n1"), uid(t, "o1"), userID)
		require.NoError(t, repo.Save(ctx, n))
		err := svc.DeleteNotification(ctx, n.ID, uid(t, "u2"))
		require.ErrorIs(t, err, domain.ErrNotificationNotFound)
		_, err = repo.GetByIDForUser(ctx, n.ID, userID)
		require.NoError(t, err)
	})

	t.Run("The notification is deleted", func(t *testing.T) {
		userID := uid(t, "u1b")
		n := mustNotification(t, uid(t, "n1"), uid(t, "o1"), userID)
		require.NoError(t, repo.Save(ctx, n))
		err := svc.DeleteNotification(ctx, n.ID, userID)
		require.NoError(t, err)
		_, err = repo.GetByIDForUser(ctx, n.ID, userID)
		assert.ErrorIs(t, err, domain.ErrNotificationNotFound)
	})
}
