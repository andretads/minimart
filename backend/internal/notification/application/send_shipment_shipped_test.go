package application

import (
	"errors"
	"minimart/internal/notification/domain"
	"minimart/internal/notification/infrastructure/memory"
	"minimart/pkg/metrics"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSendShipmentShipped(t *testing.T) {
	t.Run("Empty order id is rejected", func(t *testing.T) {
		repo := memory.New()
		svc := NewService(repo, repo, repo, newFakeEmailSender(), nil, nil, newFakeRecipientResolver(), nil, metrics.Noop{}, RetryConfig{})
		err := svc.SendShipmentShipped(t.Context(), "", "u1", "TRACK123", "Correios")
		require.ErrorIs(t, err, domain.ErrEmptyOrderID)
	})

	t.Run("Empty user_id is rejected", func(t *testing.T) {
		repo := memory.New()
		svc := NewService(repo, repo, repo, newFakeEmailSender(), nil, nil, newFakeRecipientResolver(), nil, metrics.Noop{}, RetryConfig{})
		err := svc.SendShipmentShipped(t.Context(), "o1", "", "TRACK123", "Correios")
		require.ErrorIs(t, err, domain.ErrEmptyUserID)
	})

	t.Run("Notification storage failure is propagated", func(t *testing.T) {
		boom := errors.New("disk on fire")
		resolver := newFakeRecipientResolver()
		resolver.SetEmail("u1", "user@example.com")
		svc := NewService(failingRepo{err: boom}, notificationRepoStub{}, notificationRepoStub{}, newFakeEmailSender(), nil, nil, resolver, nil, metrics.Noop{}, RetryConfig{})
		err := svc.SendShipmentShipped(t.Context(), "o1", "u1", "TRACK123", "Correios")
		require.ErrorIs(t, err, boom)
	})

	t.Run("The shipment dispatch is recorded and queued for delivery", func(t *testing.T) {
		repo := memory.New()
		resolver := newFakeRecipientResolver()
		resolver.SetEmail("u1", "user@example.com")
		prefs, _ := domain.NewPreferences("u1", true, false, false, false)
		require.NoError(t, repo.Upsert(t.Context(), prefs))
		svc := NewService(repo, repo, repo, newFakeEmailSender(), nil, nil, resolver, nil, metrics.Noop{}, RetryConfig{})

		err := svc.SendShipmentShipped(t.Context(), "o1", "u1", "TRACK123", "Correios")
		require.NoError(t, err)
		n, err := repo.GetByOrderIDAndType(t.Context(), "o1", domain.TypeShipmentShipped)
		require.NoError(t, err)
		assert.Equal(t, "o1/shipment_shipped", n.DedupeKey)
		assert.Contains(t, n.Body, "TRACK123")
		assert.Contains(t, n.Body, "Correios")
		assert.False(t, n.SentAt.IsZero())
	})

	t.Run("A shipment without tracking details is still recorded", func(t *testing.T) {
		repo := memory.New()
		resolver := newFakeRecipientResolver()
		resolver.SetEmail("u1", "user@example.com")
		prefs, _ := domain.NewPreferences("u1", true, false, false, false)
		require.NoError(t, repo.Upsert(t.Context(), prefs))
		svc := NewService(repo, repo, repo, newFakeEmailSender(), nil, nil, resolver, nil, metrics.Noop{}, RetryConfig{})

		err := svc.SendShipmentShipped(t.Context(), "o1", "u1", "", "")
		require.NoError(t, err)
		n, err := repo.GetByOrderIDAndType(t.Context(), "o1", domain.TypeShipmentShipped)
		require.NoError(t, err)
		// "tracking details" not mentioned
		assert.NotContains(t, strings.ToLower(n.Body), "rastreamento")
	})

	t.Run("Redelivered event does not duplicate the notification", func(t *testing.T) {
		repo := memory.New()
		resolver := newFakeRecipientResolver()
		resolver.SetEmail("u1", "user@example.com")
		prefs, _ := domain.NewPreferences("u1", true, false, false, false)
		require.NoError(t, repo.Upsert(t.Context(), prefs))
		svc := NewService(repo, repo, repo, newFakeEmailSender(), nil, nil, resolver, nil, metrics.Noop{}, RetryConfig{})
		require.NoError(t, svc.SendShipmentShipped(t.Context(), "o1", "u1", "TRACK123", "Correios"))
		require.NoError(t, svc.SendShipmentShipped(t.Context(), "o1", "u1", "TRACK123", "Correios"))
		list, _ := repo.ListByUserID(t.Context(), "u1", 1, 20)
		assert.Len(t, list, 1)
	})

	t.Run("Email disabled and no other contact suppresses the notification", func(t *testing.T) {
		repo := memory.New()
		resolver := newFakeRecipientResolver()
		resolver.SetEmail("u1", "user@example.com")
		prefs, _ := domain.NewPreferences("u1", false, false, false, false)
		require.NoError(t, repo.Upsert(t.Context(), prefs))
		svc := NewService(repo, repo, repo, newFakeEmailSender(), nil, nil, resolver, nil, metrics.Noop{}, RetryConfig{})
		require.NoError(t, svc.SendShipmentShipped(t.Context(), "o1", "u1", "TRACK123", "Correios"))
		_, err := repo.GetByOrderIDAndType(t.Context(), "o1", domain.TypeShipmentShipped)
		assert.Error(t, err)
	})

	t.Run("Preferences storage unavailable, so the notification is still sent", func(t *testing.T) {
		repo := memory.New()
		resolver := newFakeRecipientResolver()
		resolver.SetEmail("u1", "user@example.com")
		svc := NewService(repo, repo, failingPrefsRepo{err: errors.New("mongo down")}, newFakeEmailSender(), nil, nil, resolver, nil, metrics.Noop{}, RetryConfig{})
		require.NoError(t, svc.SendShipmentShipped(t.Context(), "o1", "u1", "TRACK123", "Correios"))
		_, err := repo.GetByOrderIDAndType(t.Context(), "o1", domain.TypeShipmentShipped)
		require.NoError(t, err)
	})

	t.Run("User has no contact on file, so the notification is suppressed", func(t *testing.T) {
		repo := memory.New()
		svc := NewService(repo, repo, repo, newFakeEmailSender(), nil, nil, newFakeRecipientResolver(), nil, metrics.Noop{}, RetryConfig{})
		require.NoError(t, svc.SendShipmentShipped(t.Context(), "o1", "u1", "TRACK123", "Correios"))
		_, err := repo.GetByOrderIDAndType(t.Context(), "o1", domain.TypeShipmentShipped)
		assert.Error(t, err)
	})
}
