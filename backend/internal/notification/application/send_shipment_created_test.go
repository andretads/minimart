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

func TestSendShipmentCreated(t *testing.T) {
	t.Run("Empty order id is rejected", func(t *testing.T) {
		repo := memory.New()
		svc := NewService(repo, repo, repo, newFakeEmailSender(), nil, nil, newFakeRecipientResolver(), nil, metrics.Noop{}, RetryConfig{})
		err := svc.SendShipmentCreated(t.Context(), "", "u1", 599)
		require.ErrorIs(t, err, domain.ErrEmptyOrderID)
	})

	t.Run("Empty user_id is rejected", func(t *testing.T) {
		repo := memory.New()
		svc := NewService(repo, repo, repo, newFakeEmailSender(), nil, nil, newFakeRecipientResolver(), nil, metrics.Noop{}, RetryConfig{})
		err := svc.SendShipmentCreated(t.Context(), "o1", "", 599)
		require.ErrorIs(t, err, domain.ErrEmptyUserID)
	})

	t.Run("Negative shipping cost is rejected", func(t *testing.T) {
		repo := memory.New()
		svc := NewService(repo, repo, repo, newFakeEmailSender(), nil, nil, newFakeRecipientResolver(), nil, metrics.Noop{}, RetryConfig{})
		err := svc.SendShipmentCreated(t.Context(), "o1", "u1", -1)
		require.ErrorIs(t, err, domain.ErrNegativeShippingCost)
	})

	t.Run("Notification storage failure is propagated", func(t *testing.T) {
		boom := errors.New("disk on fire")
		resolver := newFakeRecipientResolver()
		resolver.SetEmail("u1", "user@example.com")
		svc := NewService(failingRepo{err: boom}, notificationRepoStub{}, notificationRepoStub{}, newFakeEmailSender(), nil, nil, resolver, nil, metrics.Noop{}, RetryConfig{})
		err := svc.SendShipmentCreated(t.Context(), "o1", "u1", 599)
		require.ErrorIs(t, err, boom)
	})

	t.Run("The shipment creation is recorded and queued for delivery", func(t *testing.T) {
		repo := memory.New()
		resolver := newFakeRecipientResolver()
		resolver.SetEmail("u1", "user@example.com")
		prefs, _ := domain.NewPreferences("u1", true, false, false, false)
		require.NoError(t, repo.Upsert(t.Context(), prefs))
		svc := NewService(repo, repo, repo, newFakeEmailSender(), nil, nil, resolver, nil, metrics.Noop{}, RetryConfig{})

		err := svc.SendShipmentCreated(t.Context(), "o1", "u1", 599)
		require.NoError(t, err)
		n, err := repo.GetByOrderIDAndType(t.Context(), "o1", domain.TypeShipmentCreated)
		require.NoError(t, err)
		assert.Equal(t, "o1/shipment_created", n.DedupeKey)
		assert.Contains(t, n.Subject, "o1")
		assert.Contains(t, n.Body, "5,99")
		assert.False(t, n.SentAt.IsZero())
	})

	t.Run("Free shipping is accepted", func(t *testing.T) {
		repo := memory.New()
		resolver := newFakeRecipientResolver()
		resolver.SetEmail("u1", "user@example.com")
		prefs, _ := domain.NewPreferences("u1", true, false, false, false)
		require.NoError(t, repo.Upsert(t.Context(), prefs))
		svc := NewService(repo, repo, repo, newFakeEmailSender(), nil, nil, resolver, nil, metrics.Noop{}, RetryConfig{})

		err := svc.SendShipmentCreated(t.Context(), "o1", "u1", 0)
		require.NoError(t, err)
		_, err = repo.GetByOrderIDAndType(t.Context(), "o1", domain.TypeShipmentCreated)
		require.NoError(t, err)
	})

	t.Run("Redelivered event does not duplicate the notification", func(t *testing.T) {
		repo := memory.New()
		resolver := newFakeRecipientResolver()
		resolver.SetEmail("u1", "user@example.com")
		prefs, _ := domain.NewPreferences("u1", true, false, false, false)
		require.NoError(t, repo.Upsert(t.Context(), prefs))
		svc := NewService(repo, repo, repo, newFakeEmailSender(), nil, nil, resolver, nil, metrics.Noop{}, RetryConfig{})
		require.NoError(t, svc.SendShipmentCreated(t.Context(), "o1", "u1", 599))
		require.NoError(t, svc.SendShipmentCreated(t.Context(), "o1", "u1", 599))
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
		require.NoError(t, svc.SendShipmentCreated(t.Context(), "o1", "u1", 599))
		_, err := repo.GetByOrderIDAndType(t.Context(), "o1", domain.TypeShipmentCreated)
		assert.Error(t, err)
	})

	t.Run("Preferences storage unavailable, so the notification is still sent", func(t *testing.T) {
		repo := memory.New()
		resolver := newFakeRecipientResolver()
		resolver.SetEmail("u1", "user@example.com")
		svc := NewService(repo, repo, failingPrefsRepo{err: errors.New("mongo down")}, newFakeEmailSender(), nil, nil, resolver, nil, metrics.Noop{}, RetryConfig{})
		require.NoError(t, svc.SendShipmentCreated(t.Context(), "o1", "u1", 599))
		_, err := repo.GetByOrderIDAndType(t.Context(), "o1", domain.TypeShipmentCreated)
		require.NoError(t, err)
	})

	t.Run("User has no contact on file, so the notification is suppressed", func(t *testing.T) {
		repo := memory.New()
		svc := NewService(repo, repo, repo, newFakeEmailSender(), nil, nil, newFakeRecipientResolver(), nil, metrics.Noop{}, RetryConfig{})
		require.NoError(t, svc.SendShipmentCreated(t.Context(), "o1", "u1", 599))
		_, err := repo.GetByOrderIDAndType(t.Context(), "o1", domain.TypeShipmentCreated)
		assert.Error(t, err)
	})
}
