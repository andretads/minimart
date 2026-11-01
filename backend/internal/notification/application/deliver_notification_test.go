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

type deliveryEmailSender struct {
	err error
}

func (s *deliveryEmailSender) Send(ctx context.Context, e domain.Email) (domain.Receipt, error) {
	if s.err != nil {
		return domain.Receipt{}, s.err
	}
	return domain.Receipt{ProviderMessageID: "msg-1", Delivered: false}, nil
}

func TestDeliverNotification(t *testing.T) {
	setup := func(t *testing.T, sendErr error) (*Service, string, string) {
		t.Helper()
		repo := memory.New()
		prefs, _ := domain.NewPreferences("u-1", true, false, false, false)
		require.NoError(t, repo.Upsert(t.Context(), prefs))
		resolver := newFakeRecipientResolver()
		resolver.SetEmail("u-1", "user@example.com")
		sender := &deliveryEmailSender{err: sendErr}
		svc := NewService(repo, repo, repo, sender, nil, nil, resolver, nil, metrics.Noop{}, RetryConfig{MaxAttempts: 3, BaseDelay: time.Millisecond, MaxDelay: time.Second})

		n, err := domain.NewOrderConfirmation(newID(), "o-1", "u-1", 1250)
		require.NoError(t, err)
		_, deliveries, err := svc.IngestNotification(t.Context(), n, nil)
		require.NoError(t, err)
		require.Len(t, deliveries, 1)
		return svc, deliveries[0].ID, n.ID
	}

	t.Run("marks delivery sent on success", func(t *testing.T) {
		svc, deliveryID, notificationID := setup(t, nil)
		require.NoError(t, svc.DeliverNotification(t.Context(), deliveryID))

		d, err := svc.deliveries.GetDeliveryByID(t.Context(), deliveryID)
		require.NoError(t, err)
		assert.Equal(t, domain.DeliveryStatusSent, d.Status)
		assert.Equal(t, "msg-1", d.ProviderMessageID)

		ns, err := svc.repo.GetByID(t.Context(), notificationID)
		require.NoError(t, err)
		assert.Equal(t, domain.NotificationStatusSent, ns.Status)
	})

	t.Run("schedules retry on transient error", func(t *testing.T) {
		svc, deliveryID, _ := setup(t, domain.ProviderError{Transient: true, Err: errors.New("timeout")})
		require.NoError(t, svc.DeliverNotification(t.Context(), deliveryID))

		d, err := svc.deliveries.GetDeliveryByID(t.Context(), deliveryID)
		require.NoError(t, err)
		assert.Equal(t, domain.DeliveryStatusRetryPending, d.Status)
		assert.NotNil(t, d.NextAttemptAt)
	})

	t.Run("fails permanently on permanent error", func(t *testing.T) {
		svc, deliveryID, _ := setup(t, domain.ProviderError{Permanent: true, Err: errors.New("bad address")})
		err := svc.DeliverNotification(t.Context(), deliveryID)
		require.Error(t, err)
		assert.True(t, kafka.IsPermanent(err))

		d, dErr := svc.deliveries.GetDeliveryByID(t.Context(), deliveryID)
		require.NoError(t, dErr)
		assert.Equal(t, domain.DeliveryStatusFailed, d.Status)
	})

	t.Run("terminal delivery is not reprocessed", func(t *testing.T) {
		svc, deliveryID, _ := setup(t, nil)
		require.NoError(t, svc.DeliverNotification(t.Context(), deliveryID))
		require.NoError(t, svc.DeliverNotification(t.Context(), deliveryID))

		d, err := svc.deliveries.GetDeliveryByID(t.Context(), deliveryID)
		require.NoError(t, err)
		assert.Equal(t, domain.DeliveryStatusSent, d.Status)
		assert.Equal(t, 1, d.AttemptCount)
	})
}
