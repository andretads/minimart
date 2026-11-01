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
	t.Run("A construction error is propagated", func(t *testing.T) {
		repo := memory.New()
		svc := NewService(repo, repo, repo, newFakeEmailSender(), nil, nil, newFakeRecipientResolver(), nil, metrics.Noop{}, RetryConfig{})

		_, _, err := svc.IngestNotification(t.Context(), domain.Notification{}, errors.New("bad notification"))
		require.Error(t, err)
		assert.EqualError(t, err, "bad notification")
	})

	t.Run("Empty notification id is rejected", func(t *testing.T) {
		repo := memory.New()
		svc := NewService(repo, repo, repo, newFakeEmailSender(), nil, nil, newFakeRecipientResolver(), nil, metrics.Noop{}, RetryConfig{})

		n := domain.Notification{ID: "", Type: domain.TypeOrderConfirmation, OrderID: "o1", UserID: "u1"}
		_, _, err := svc.IngestNotification(t.Context(), n, nil)
		require.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrEmptyID)
	})

	t.Run("Empty user_id is rejected", func(t *testing.T) {
		repo := memory.New()
		svc := NewService(repo, repo, repo, newFakeEmailSender(), nil, nil, newFakeRecipientResolver(), nil, metrics.Noop{}, RetryConfig{})

		n := domain.Notification{ID: "n1", Type: domain.TypeOrderConfirmation, OrderID: "o1", UserID: ""}
		_, _, err := svc.IngestNotification(t.Context(), n, nil)
		require.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrEmptyUserID)
	})

	t.Run("Missing order id and dedupe key is rejected", func(t *testing.T) {
		repo := memory.New()
		svc := NewService(repo, repo, repo, newFakeEmailSender(), nil, nil, newFakeRecipientResolver(), nil, metrics.Noop{}, RetryConfig{})

		n := domain.Notification{ID: "n1", Type: domain.TypeOrderConfirmation, OrderID: "", UserID: "u1", DedupeKey: ""}
		_, _, err := svc.IngestNotification(t.Context(), n, nil)
		require.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrEmptyOrderID)
	})

	t.Run("Defaults are filled in for a valid notification", func(t *testing.T) {
		repo := memory.New()
		resolver := newFakeRecipientResolver()
		resolver.SetEmail("u1", "user@example.com")
		prefs, _ := domain.NewPreferences("u1", true, false, false, false)
		require.NoError(t, repo.Upsert(t.Context(), prefs))
		svc := NewService(repo, repo, repo, newFakeEmailSender(), nil, nil, resolver, nil, metrics.Noop{}, RetryConfig{})

		n := domain.Notification{ID: "n1", Type: domain.TypeOrderConfirmation, OrderID: "o1", UserID: "u1"}
		got, _, err := svc.IngestNotification(t.Context(), n, nil)
		require.NoError(t, err)
		assert.Equal(t, "u1", got.RecipientID)
		assert.Equal(t, "o1/order_confirmation", got.DedupeKey)
		assert.Equal(t, "order_confirmation_v1", got.TemplateID)
		assert.Equal(t, domain.PriorityNormal, got.Priority)
		assert.Equal(t, domain.NotificationStatusPending, got.Status)
	})

	t.Run("One delivery is queued per eligible channel", func(t *testing.T) {
		repo := memory.New()
		prefs, _ := domain.NewPreferences("u1", true, true, false, false)
		require.NoError(t, repo.Upsert(t.Context(), prefs))
		resolver := newFakeRecipientResolver()
		resolver.SetEmail("u1", "user@example.com")
		resolver.SetPhone("u1", "+1")
		svc := NewService(repo, repo, repo, newFakeEmailSender(), nil, nil, resolver, nil, metrics.Noop{}, RetryConfig{})

		n := domain.Notification{ID: "n1", Type: domain.TypeOrderConfirmation, OrderID: "o1", UserID: "u1"}
		_, deliveries, err := svc.IngestNotification(t.Context(), n, nil)
		require.NoError(t, err)
		require.Len(t, deliveries, 2)

		emailFound, smsFound := false, false
		for _, d := range deliveries {
			if d.Channel == domain.ChannelEmail && d.Status == domain.DeliveryStatusPending {
				emailFound = true
			}
			if d.Channel == domain.ChannelSMS && d.Status == domain.DeliveryStatusPending {
				smsFound = true
			}
		}
		assert.True(t, emailFound)
		assert.True(t, smsFound)
	})

	t.Run("A notification with no eligible channel is suppressed and not stored", func(t *testing.T) {
		repo := memory.New()
		prefs, _ := domain.NewPreferences("u1", false, false, false, false)
		require.NoError(t, repo.Upsert(t.Context(), prefs))
		resolver := newFakeRecipientResolver()
		resolver.SetEmail("u1", "user@example.com")
		svc := NewService(repo, repo, repo, newFakeEmailSender(), nil, nil, resolver, nil, metrics.Noop{}, RetryConfig{})

		n := domain.Notification{ID: "n1", Type: domain.TypeOrderConfirmation, OrderID: "o1", UserID: "u1"}
		got, deliveries, err := svc.IngestNotification(t.Context(), n, nil)
		require.NoError(t, err)
		assert.Equal(t, domain.NotificationStatusSuppressed, got.Status)
		assert.Nil(t, deliveries)

		_, err = repo.GetByOrderIDAndType(t.Context(), "o1", domain.TypeOrderConfirmation)
		assert.Error(t, err)
	})

	t.Run("A duplicate dedupe key reuses the stored notification and its deliveries", func(t *testing.T) {
		repo := memory.New()
		prefs, _ := domain.NewPreferences("u1", true, false, false, false)
		require.NoError(t, repo.Upsert(t.Context(), prefs))
		resolver := newFakeRecipientResolver()
		resolver.SetEmail("u1", "user@example.com")
		svc := NewService(repo, repo, repo, newFakeEmailSender(), nil, nil, resolver, nil, metrics.Noop{}, RetryConfig{})

		n1 := domain.Notification{ID: "n1", Type: domain.TypeOrderConfirmation, OrderID: "o1", UserID: "u1"}
		_, _, err := svc.IngestNotification(t.Context(), n1, nil)
		require.NoError(t, err)

		n2 := domain.Notification{ID: "n2", Type: domain.TypeOrderConfirmation, OrderID: "o1", UserID: "u1"}
		got, deliveries, err := svc.IngestNotification(t.Context(), n2, nil)
		require.NoError(t, err)
		assert.Equal(t, "n1", got.ID)

		all, err := repo.ListByUserID(t.Context(), "u1", 1, 20)
		require.NoError(t, err)
		assert.Len(t, all, 1)

		// deliveries reused
		assert.Len(t, deliveries, 1)
	})

	t.Run("A transactional notification is sent when preferences are unavailable", func(t *testing.T) {
		boom := errors.New("mongo down")
		// prefs failing but use failingPrefsRepo for prefs
		repo := memory.New()
		resolver := newFakeRecipientResolver()
		resolver.SetEmail("u1", "user@example.com")
		svc := NewService(repo, repo, failingPrefsRepo{err: boom}, newFakeEmailSender(), nil, nil, resolver, nil, metrics.Noop{}, RetryConfig{})

		// Need a logger? no. Check warn logged not asserted strictly.
		n := domain.Notification{ID: "n1", Type: domain.TypeOrderConfirmation, OrderID: "o1", UserID: "u1"}
		got, deliveries, err := svc.IngestNotification(t.Context(), n, nil)
		require.NoError(t, err)
		assert.Equal(t, domain.TypeOrderConfirmation, got.Type)
		assert.Equal(t, "o1", got.OrderID)
		assert.Equal(t, "u1", got.UserID)
		assert.Equal(t, domain.NotificationStatusPending, got.Status)
		require.Len(t, deliveries, 1)
	})
}
