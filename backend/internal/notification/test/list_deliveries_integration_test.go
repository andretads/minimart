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

func TestListDeliveriesIntegration(t *testing.T) {
	ctx := t.Context()
	repo := newTestRepo(t)
	deliveryRepo := newTestDeliveryRepo(t)
	prefsRepo := newTestPrefsRepo(t)
	resolver := newFakeRecipientResolver()
	svc := application.NewService(repo, deliveryRepo, prefsRepo, newFakeEmailSender(), nil, nil, resolver, nil, metrics.Noop{}, application.RetryConfig{})

	t.Run("Empty notification id is rejected", func(t *testing.T) {
		_, err := svc.ListDeliveries(ctx, "", "u1")
		require.ErrorIs(t, err, domain.ErrEmptyNotificationID)
	})

	t.Run("Notification does not exist", func(t *testing.T) {
		_, err := svc.ListDeliveries(ctx, uid(t, "missing"), "u1")
		require.ErrorIs(t, err, domain.ErrNotificationNotFound)
	})

	t.Run("Another user's notification is not found", func(t *testing.T) {
		userID := uid(t, "u1")
		resolver.SetEmail(userID, "user@example.com")
		prefs, _ := domain.NewPreferences(userID, true, false, false, false)
		require.NoError(t, prefsRepo.Upsert(ctx, prefs))
		n, err := domain.NewOrderConfirmation(uid(t, "o1"), uid(t, "o1id"), userID, 1250)
		require.NoError(t, err)
		_, _, err = svc.IngestNotification(ctx, n, nil)
		require.NoError(t, err)
		_, err = svc.ListDeliveries(ctx, n.ID, uid(t, "u2"))
		require.ErrorIs(t, err, domain.ErrNotificationNotFound)
	})

	t.Run("The notification's deliveries are returned", func(t *testing.T) {
		userID := uid(t, "u1b")
		resolver.SetEmail(userID, "user@example.com")
		prefs, _ := domain.NewPreferences(userID, true, false, false, false)
		require.NoError(t, prefsRepo.Upsert(ctx, prefs))
		n, err := domain.NewOrderConfirmation(uid(t, "o2"), uid(t, "o2id"), userID, 1250)
		require.NoError(t, err)
		_, _, err = svc.IngestNotification(ctx, n, nil)
		require.NoError(t, err)
		got, err := svc.ListDeliveries(ctx, n.ID, userID)
		require.NoError(t, err)
		require.Len(t, got, 1)
		assert.Equal(t, domain.DeliveryStatusPending, got[0].Status)
	})
}
