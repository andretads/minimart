package application

import (
	"context"
	"errors"
	"minimart/internal/notification/domain"
	"minimart/internal/notification/infrastructure/memory"
	"minimart/pkg/kafka"
	"minimart/pkg/metrics"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type deliverEmailSender struct {
	err       error
	delivered bool
}

func (s *deliverEmailSender) Send(ctx context.Context, e domain.Email) (domain.Receipt, error) {
	if s.err != nil {
		return domain.Receipt{}, s.err
	}
	return domain.Receipt{ProviderMessageID: "msg-1", Delivered: s.delivered}, nil
}

func TestDeliverNotification(t *testing.T) {
	t.Run("Delivery does not exist", func(t *testing.T) {
		repo := memory.New()
		svc := NewService(repo, repo, repo, &deliverEmailSender{}, nil, nil, newFakeRecipientResolver(), nil, metrics.Noop{}, RetryConfig{})
		err := svc.DeliverNotification(t.Context(), "missing")
		assert.ErrorIs(t, err, domain.ErrDeliveryNotFound)
	})

	t.Run("Notification does not exist", func(t *testing.T) {
		repo := memory.New()
		// create a delivery pointing to non-existent notification
		d, _ := domain.NewDelivery("d1", "nonexistent", "u1", domain.ChannelEmail, "resend")
		require.NoError(t, repo.SaveDelivery(t.Context(), d))
		svc := NewService(repo, repo, repo, &deliverEmailSender{}, nil, nil, newFakeRecipientResolver(), nil, metrics.Noop{}, RetryConfig{})
		err := svc.DeliverNotification(t.Context(), "d1")
		assert.ErrorIs(t, err, domain.ErrNotificationNotFound)
	})

	setup := func(t *testing.T, sendErr error, delivered bool, maxAttempts int) (*Service, string, string) {
		t.Helper()
		repo := memory.New()
		prefs, _ := domain.NewPreferences("u1", true, false, false, false)
		require.NoError(t, repo.Upsert(t.Context(), prefs))
		resolver := newFakeRecipientResolver()
		resolver.SetEmail("u1", "user@example.com")
		sender := &deliverEmailSender{err: sendErr, delivered: delivered}
		if maxAttempts <= 0 {
			maxAttempts = 3
		}
		svc := NewService(repo, repo, repo, sender, nil, nil, resolver, nil, metrics.Noop{}, RetryConfig{MaxAttempts: maxAttempts, BaseDelay: time.Millisecond, MaxDelay: time.Second})
		n, err := domain.NewOrderConfirmation(newID(), "o1", "u1", 1250)
		require.NoError(t, err)
		_, deliveries, err := svc.IngestNotification(t.Context(), n, nil)
		require.NoError(t, err)
		require.Len(t, deliveries, 1)
		return svc, deliveries[0].ID, n.ID
	}

	t.Run("A successful send marks the delivery and notification as sent", func(t *testing.T) {
		svc, deliveryID, notificationID := setup(t, nil, false, 3)
		require.NoError(t, svc.DeliverNotification(t.Context(), deliveryID))
		d, err := svc.deliveries.GetDeliveryByID(t.Context(), deliveryID)
		require.NoError(t, err)
		assert.Equal(t, domain.DeliveryStatusSent, d.Status)
		assert.NotEmpty(t, d.ProviderMessageID)
		ns, err := svc.repo.GetByID(t.Context(), notificationID)
		require.NoError(t, err)
		assert.Equal(t, domain.NotificationStatusSent, ns.Status)
	})

	t.Run("A confirmed delivery marks the delivery and notification as delivered", func(t *testing.T) {
		svc, deliveryID, notificationID := setup(t, nil, true, 3)
		require.NoError(t, svc.DeliverNotification(t.Context(), deliveryID))
		d, err := svc.deliveries.GetDeliveryByID(t.Context(), deliveryID)
		require.NoError(t, err)
		assert.Equal(t, domain.DeliveryStatusDelivered, d.Status)
		ns, err := svc.repo.GetByID(t.Context(), notificationID)
		require.NoError(t, err)
		assert.Equal(t, domain.NotificationStatusDelivered, ns.Status)
	})

	t.Run("A transient provider error schedules a retry", func(t *testing.T) {
		svc, deliveryID, _ := setup(t, domain.ProviderError{Transient: true, Err: errors.New("timeout")}, false, 3)
		require.NoError(t, svc.DeliverNotification(t.Context(), deliveryID))
		d, err := svc.deliveries.GetDeliveryByID(t.Context(), deliveryID)
		require.NoError(t, err)
		assert.Equal(t, domain.DeliveryStatusRetryPending, d.Status)
		assert.NotNil(t, d.NextAttemptAt)
	})

	t.Run("A permanent provider error fails the delivery permanently", func(t *testing.T) {
		svc, deliveryID, _ := setup(t, domain.ProviderError{Permanent: true, Err: errors.New("bad address")}, false, 3)
		err := svc.DeliverNotification(t.Context(), deliveryID)
		require.Error(t, err)
		assert.True(t, kafka.IsPermanent(err))
		d, dErr := svc.deliveries.GetDeliveryByID(t.Context(), deliveryID)
		require.NoError(t, dErr)
		assert.Equal(t, domain.DeliveryStatusFailed, d.Status)
	})

	t.Run("An exhausted retry budget fails the delivery permanently", func(t *testing.T) {
		svc, deliveryID, _ := setup(t, domain.ProviderError{Transient: true, Err: errors.New("timeout")}, false, 1)
		err := svc.DeliverNotification(t.Context(), deliveryID)
		require.Error(t, err)
		assert.True(t, kafka.IsPermanent(err))
		d, dErr := svc.deliveries.GetDeliveryByID(t.Context(), deliveryID)
		require.NoError(t, dErr)
		assert.Equal(t, domain.DeliveryStatusFailed, d.Status)
	})

	t.Run("A terminal delivery is not processed again", func(t *testing.T) {
		svc, deliveryID, _ := setup(t, nil, false, 3)
		require.NoError(t, svc.DeliverNotification(t.Context(), deliveryID))
		require.NoError(t, svc.DeliverNotification(t.Context(), deliveryID))
		d, err := svc.deliveries.GetDeliveryByID(t.Context(), deliveryID)
		require.NoError(t, err)
		assert.Equal(t, domain.DeliveryStatusSent, d.Status)
		assert.Equal(t, 1, d.AttemptCount)
	})
}
