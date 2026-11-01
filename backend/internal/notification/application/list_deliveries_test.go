package application

import (
	"minimart/internal/notification/domain"
	"minimart/internal/notification/infrastructure/memory"
	"minimart/pkg/metrics"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestListDeliveries(t *testing.T) {
	t.Run("Empty notification id is rejected", func(t *testing.T) {
		repo := memory.New()
		prefs, _ := domain.NewPreferences("u1", true, false, false, false)
		require.NoError(t, repo.Upsert(t.Context(), prefs))
		resolver := newFakeRecipientResolver()
		resolver.SetEmail("u1", "user@example.com")
		svc := NewService(repo, repo, repo, newFakeEmailSender(), nil, nil, resolver, nil, metrics.Noop{}, RetryConfig{})

		_, err := svc.ListDeliveries(t.Context(), "", "u1")
		require.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrEmptyNotificationID)
	})

	t.Run("Notification does not exist", func(t *testing.T) {
		repo := memory.New()
		svc := NewService(repo, repo, repo, newFakeEmailSender(), nil, nil, newFakeRecipientResolver(), nil, metrics.Noop{}, RetryConfig{})

		_, err := svc.ListDeliveries(t.Context(), "missing", "u1")
		require.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrNotificationNotFound)
	})

	t.Run("Another user's notification is not found", func(t *testing.T) {
		repo := memory.New()
		prefs, _ := domain.NewPreferences("u1", true, false, false, false)
		require.NoError(t, repo.Upsert(t.Context(), prefs))
		resolver := newFakeRecipientResolver()
		resolver.SetEmail("u1", "user@example.com")
		svc := NewService(repo, repo, repo, newFakeEmailSender(), nil, nil, resolver, nil, metrics.Noop{}, RetryConfig{})

		n, err := domain.NewOrderConfirmation(newID(), "o1", "u1", 1250)
		require.NoError(t, err)
		_, _, err = svc.IngestNotification(t.Context(), n, nil)
		require.NoError(t, err)

		_, err = svc.ListDeliveries(t.Context(), n.ID, "u2")
		require.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrNotificationNotFound)
	})

	t.Run("The notification's deliveries are returned", func(t *testing.T) {
		repo := memory.New()
		prefs, _ := domain.NewPreferences("u1", true, false, false, false)
		require.NoError(t, repo.Upsert(t.Context(), prefs))
		resolver := newFakeRecipientResolver()
		resolver.SetEmail("u1", "user@example.com")
		svc := NewService(repo, repo, repo, newFakeEmailSender(), nil, nil, resolver, nil, metrics.Noop{}, RetryConfig{})

		n, err := domain.NewOrderConfirmation(newID(), "o1", "u1", 1250)
		require.NoError(t, err)
		_, deliveries, err := svc.IngestNotification(t.Context(), n, nil)
		require.NoError(t, err)
		require.Len(t, deliveries, 1)

		got, err := svc.ListDeliveries(t.Context(), n.ID, "u1")
		require.NoError(t, err)
		require.Len(t, got, 1)
		assert.Equal(t, domain.DeliveryStatusPending, got[0].Status)
		assert.Equal(t, domain.ChannelEmail, got[0].Channel)
	})
}
