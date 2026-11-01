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

func TestIngestNotification(t *testing.T) {
	t.Run("queues one delivery per eligible channel", func(t *testing.T) {
		repo := memory.New()
		prefs, _ := domain.NewPreferences("u-1", true, true, true, true)
		require.NoError(t, repo.Upsert(t.Context(), prefs))
		resolver := newFakeRecipientResolver()
		resolver.SetEmail("u-1", "user@example.com")

		svc := NewService(repo, repo, repo, newFakeEmailSender(), nil, nil, resolver, nil, metrics.Noop{}, RetryConfig{})

		n, err := domain.NewOrderConfirmation(newID(), "o-1", "u-1", 1250)
		require.NoError(t, err)
		got, deliveries, err := svc.IngestNotification(t.Context(), n, nil)
		require.NoError(t, err)
		assert.Equal(t, domain.NotificationStatusPending, got.Status)
		require.Len(t, deliveries, 1)
		assert.Equal(t, domain.ChannelEmail, deliveries[0].Channel)
	})

	t.Run("suppresses when no channel is eligible", func(t *testing.T) {
		repo := memory.New()
		prefs, _ := domain.NewPreferences("u-1", false, false, false, false)
		require.NoError(t, repo.Upsert(t.Context(), prefs))
		resolver := newFakeRecipientResolver()
		resolver.SetEmail("u-1", "user@example.com")
		svc := NewService(repo, repo, repo, newFakeEmailSender(), nil, nil, resolver, nil, metrics.Noop{}, RetryConfig{})

		n, err := domain.NewOrderConfirmation(newID(), "o-1", "u-1", 1250)
		require.NoError(t, err)
		got, deliveries, err := svc.IngestNotification(t.Context(), n, nil)
		require.NoError(t, err)
		assert.Equal(t, domain.NotificationStatusSuppressed, got.Status)
		assert.Empty(t, deliveries)
	})

	t.Run("construction error is propagated", func(t *testing.T) {
		repo := memory.New()
		resolver := newFakeRecipientResolver()
		svc := NewService(repo, repo, repo, newFakeEmailSender(), nil, nil, resolver, nil, metrics.Noop{}, RetryConfig{})

		_, _, err := svc.IngestNotification(t.Context(), domain.Notification{}, errors.New("bad"))
		require.Error(t, err)
	})

	t.Run("idempotent re-ingest does not duplicate deliveries", func(t *testing.T) {
		repo := memory.New()
		prefs, _ := domain.NewPreferences("u-1", true, false, false, false)
		require.NoError(t, repo.Upsert(t.Context(), prefs))
		resolver := newFakeRecipientResolver()
		resolver.SetEmail("u-1", "user@example.com")
		svc := NewService(repo, repo, repo, newFakeEmailSender(), nil, nil, resolver, nil, metrics.Noop{}, RetryConfig{})

		n, err := domain.NewOrderConfirmation(newID(), "o-1", "u-1", 1250)
		require.NoError(t, err)
		_, _, err = svc.IngestNotification(t.Context(), n, nil)
		require.NoError(t, err)
		_, _, err = svc.IngestNotification(t.Context(), n, nil)
		require.NoError(t, err)

		ds, err := repo.ListByNotification(t.Context(), n.ID)
		require.NoError(t, err)
		assert.Len(t, ds, 1)
	})

	t.Run("duplicate event reuses the persisted notification", func(t *testing.T) {
		repo := memory.New()
		prefs, _ := domain.NewPreferences("u-1", true, true, false, false)
		require.NoError(t, repo.Upsert(t.Context(), prefs))
		resolver := newFakeRecipientResolver()
		resolver.SetEmail("u-1", "user@example.com")
		resolver.SetPhone("u-1", "+5511999999999")
		svc := NewService(repo, repo, repo, newFakeEmailSender(), nil, nil, resolver, nil, metrics.Noop{}, RetryConfig{})

		first, err := domain.NewOrderConfirmation(newID(), "o-1", "u-1", 1250)
		require.NoError(t, err)
		persisted, _, err := svc.IngestNotification(t.Context(), first, nil)
		require.NoError(t, err)

		duplicate, err := domain.NewOrderConfirmation(newID(), "o-1", "u-1", 1250)
		require.NoError(t, err)
		got, deliveries, err := svc.IngestNotification(t.Context(), duplicate, nil)
		require.NoError(t, err)

		assert.Equal(t, persisted.ID, got.ID)
		assert.Len(t, deliveries, 2)

		orphans, err := repo.ListByNotification(t.Context(), duplicate.ID)
		require.NoError(t, err)
		assert.Empty(t, orphans, "a duplicate event must not create deliveries for an unsaved notification")
	})
}
