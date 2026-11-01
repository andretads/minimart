package application

import (
	"errors"
	"minimart/internal/notification/domain"
	"minimart/internal/notification/infrastructure/memory"
	"minimart/pkg/metrics"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSendOrderConfirmation(t *testing.T) {
	t.Run("Empty order id is rejected", func(t *testing.T) {
		repo := memory.New()
		svc := NewService(repo, repo, repo, newFakeEmailSender(), nil, nil, newFakeRecipientResolver(), nil, metrics.Noop{}, RetryConfig{})

		err := svc.SendOrderConfirmation(t.Context(), "", "u1", 1250)
		require.ErrorIs(t, err, domain.ErrEmptyOrderID)

		_, err = repo.GetByOrderIDAndType(t.Context(), "", domain.TypeOrderConfirmation)
		require.Error(t, err)
	})

	t.Run("Empty user_id is rejected", func(t *testing.T) {
		repo := memory.New()
		svc := NewService(repo, repo, repo, newFakeEmailSender(), nil, nil, newFakeRecipientResolver(), nil, metrics.Noop{}, RetryConfig{})

		err := svc.SendOrderConfirmation(t.Context(), "o1", "", 1250)
		require.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrEmptyUserID)
	})

	t.Run("Non-positive total is rejected", func(t *testing.T) {
		repo := memory.New()
		svc := NewService(repo, repo, repo, newFakeEmailSender(), nil, nil, newFakeRecipientResolver(), nil, metrics.Noop{}, RetryConfig{})

		err := svc.SendOrderConfirmation(t.Context(), "o1", "u1", 0)
		require.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrNonPositiveTotal)
	})

	t.Run("Notification storage failure is propagated", func(t *testing.T) {
		boom := errors.New("disk on fire")
		repo := failingRepo{err: boom}
		resolver := newFakeRecipientResolver()
		resolver.SetEmail("u1", "user@example.com")
		svc := NewService(repo, repo, repo, newFakeEmailSender(), nil, nil, resolver, nil, metrics.Noop{}, RetryConfig{})
		err := svc.SendOrderConfirmation(t.Context(), "o1", "u1", 1250)
		require.Error(t, err)
		assert.ErrorIs(t, err, boom)
	})

	t.Run("The confirmation is recorded and queued for delivery", func(t *testing.T) {
		repo := memory.New()
		resolver := newFakeRecipientResolver()
		resolver.SetEmail("u1", "user@example.com")
		prefs, _ := domain.NewPreferences("u1", true, false, false, false)
		require.NoError(t, repo.Upsert(t.Context(), prefs))
		svc := NewService(repo, repo, repo, newFakeEmailSender(), nil, nil, resolver, nil, metrics.Noop{}, RetryConfig{})

		err := svc.SendOrderConfirmation(t.Context(), "o1", "u1", 1250)
		require.NoError(t, err)

		n, err := repo.GetByOrderIDAndType(t.Context(), "o1", domain.TypeOrderConfirmation)
		require.NoError(t, err)
		assert.Equal(t, domain.TypeOrderConfirmation, n.Type)
		assert.Equal(t, "o1", n.OrderID)
		assert.Equal(t, "u1", n.UserID)
		assert.Equal(t, "o1/order_confirmation", n.DedupeKey)
		assert.Equal(t, domain.NotificationStatusPending, n.Status)
		assert.Contains(t, n.Subject, "o1")
		assert.False(t, n.SentAt.IsZero())

		deliveries, err := repo.ListByNotification(t.Context(), n.ID)
		require.NoError(t, err)
		require.Len(t, deliveries, 1)
		assert.Equal(t, domain.DeliveryStatusPending, deliveries[0].Status)
		assert.Equal(t, domain.ChannelEmail, deliveries[0].Channel)
	})

	t.Run("Every eligible channel gets its own delivery", func(t *testing.T) {
		repo := memory.New()
		resolver := newFakeRecipientResolver()
		resolver.SetEmail("u1", "user@example.com")
		resolver.SetPhone("u1", "+1")
		prefs, _ := domain.NewPreferences("u1", true, true, false, false)
		require.NoError(t, repo.Upsert(t.Context(), prefs))
		svc := NewService(repo, repo, repo, newFakeEmailSender(), nil, nil, resolver, nil, metrics.Noop{}, RetryConfig{})

		err := svc.SendOrderConfirmation(t.Context(), "o1", "u1", 1250)
		require.NoError(t, err)

		n, err := repo.GetByOrderIDAndType(t.Context(), "o1", domain.TypeOrderConfirmation)
		require.NoError(t, err)
		deliveries, err := repo.ListByNotification(t.Context(), n.ID)
		require.NoError(t, err)
		require.Len(t, deliveries, 2)

		channels := map[domain.Channel]bool{}
		for _, d := range deliveries {
			channels[d.Channel] = true
			assert.Equal(t, domain.DeliveryStatusPending, d.Status)
		}
		assert.True(t, channels[domain.ChannelEmail])
		assert.True(t, channels[domain.ChannelSMS])
	})

	t.Run("Redelivered event does not duplicate the confirmation", func(t *testing.T) {
		repo := memory.New()
		resolver := newFakeRecipientResolver()
		resolver.SetEmail("u1", "user@example.com")
		prefs, _ := domain.NewPreferences("u1", true, false, false, false)
		require.NoError(t, repo.Upsert(t.Context(), prefs))
		svc := NewService(repo, repo, repo, newFakeEmailSender(), nil, nil, resolver, nil, metrics.Noop{}, RetryConfig{})

		require.NoError(t, svc.SendOrderConfirmation(t.Context(), "o1", "u1", 1250))
		require.NoError(t, svc.SendOrderConfirmation(t.Context(), "o1", "u1", 1250))

		list, err := repo.ListByUserID(t.Context(), "u1", 1, 20)
		require.NoError(t, err)
		assert.Len(t, list, 1)

		n, err := repo.GetByOrderIDAndType(t.Context(), "o1", domain.TypeOrderConfirmation)
		require.NoError(t, err)
		deliveries, err := repo.ListByNotification(t.Context(), n.ID)
		require.NoError(t, err)
		assert.Len(t, deliveries, 1)
	})

	t.Run("Email disabled and no other contact suppresses the confirmation", func(t *testing.T) {
		repo := memory.New()
		resolver := newFakeRecipientResolver()
		resolver.SetEmail("u1", "user@example.com")
		prefs, _ := domain.NewPreferences("u1", false, false, false, false)
		require.NoError(t, repo.Upsert(t.Context(), prefs))
		svc := NewService(repo, repo, repo, newFakeEmailSender(), nil, nil, resolver, nil, metrics.Noop{}, RetryConfig{})

		err := svc.SendOrderConfirmation(t.Context(), "o1", "u1", 1250)
		require.NoError(t, err)

		_, err = repo.GetByOrderIDAndType(t.Context(), "o1", domain.TypeOrderConfirmation)
		require.Error(t, err)

		list, err := repo.ListByUserID(t.Context(), "u1", 1, 20)
		require.NoError(t, err)
		assert.Empty(t, list)
	})

	t.Run("Preferences storage unavailable, so the confirmation is still sent", func(t *testing.T) {
		repo := memory.New()
		resolver := newFakeRecipientResolver()
		resolver.SetEmail("u1", "user@example.com")
		boom := errors.New("mongo down")
		svc := NewService(repo, repo, failingPrefsRepo{err: boom}, newFakeEmailSender(), nil, nil, resolver, nil, metrics.Noop{}, RetryConfig{})

		err := svc.SendOrderConfirmation(t.Context(), "o1", "u1", 1250)
		require.NoError(t, err)

		n, err := repo.GetByOrderIDAndType(t.Context(), "o1", domain.TypeOrderConfirmation)
		require.NoError(t, err)
		assert.Equal(t, domain.NotificationStatusPending, n.Status)

		deliveries, err := repo.ListByNotification(t.Context(), n.ID)
		require.NoError(t, err)
		require.Len(t, deliveries, 1)
	})

	t.Run("User has no contact on file, so the confirmation is suppressed", func(t *testing.T) {
		repo := memory.New()
		resolver := newFakeRecipientResolver()
		svc := NewService(repo, repo, repo, newFakeEmailSender(), nil, nil, resolver, nil, metrics.Noop{}, RetryConfig{})

		err := svc.SendOrderConfirmation(t.Context(), "o1", "u1", 1250)
		require.NoError(t, err)

		_, err = repo.GetByOrderIDAndType(t.Context(), "o1", domain.TypeOrderConfirmation)
		require.Error(t, err)
	})
}
