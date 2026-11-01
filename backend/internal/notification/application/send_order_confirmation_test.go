package application

import (
	"context"
	"errors"
	"minimart/internal/notification/domain"
	"minimart/internal/notification/infrastructure/memory"
	"minimart/pkg/metrics"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type notificationRepoStub struct{}

func (notificationRepoStub) Save(context.Context, domain.Notification) error { return nil }
func (notificationRepoStub) GetByIDForUser(context.Context, string, string) (domain.Notification, error) {
	return domain.Notification{}, domain.ErrNotificationNotFound
}

func (notificationRepoStub) ListByUserID(context.Context, string, int, int) ([]domain.Notification, error) {
	return nil, nil
}

func (notificationRepoStub) GetByID(context.Context, string) (domain.Notification, error) {
	return domain.Notification{}, domain.ErrNotificationNotFound
}

func (notificationRepoStub) CountByUserID(context.Context, string) (int, error) { return 0, nil }

func (notificationRepoStub) GetByDedupeKey(context.Context, string) (domain.Notification, error) {
	return domain.Notification{}, domain.ErrNotificationNotFound
}

func (notificationRepoStub) UpdateStatus(context.Context, string, domain.NotificationStatus, *time.Time, *time.Time) error {
	return nil
}

func (notificationRepoStub) MarkRead(context.Context, string, string) error   { return nil }
func (notificationRepoStub) MarkUnread(context.Context, string, string) error { return nil }
func (notificationRepoStub) MarkAllRead(context.Context, string) error        { return nil }
func (notificationRepoStub) Delete(context.Context, string, string) error     { return nil }

func (notificationRepoStub) GetByOrderIDAndType(context.Context, string, domain.Type) (domain.Notification, error) {
	return domain.Notification{}, domain.ErrNotificationNotFound
}

func (notificationRepoStub) GetByUserID(_ context.Context, userID string) (domain.Preferences, error) {
	return domain.DefaultPreferences(userID), nil
}

func (notificationRepoStub) Upsert(context.Context, domain.Preferences) error    { return nil }
func (notificationRepoStub) SaveDelivery(context.Context, domain.Delivery) error { return nil }
func (notificationRepoStub) GetDeliveryByID(context.Context, string) (domain.Delivery, error) {
	return domain.Delivery{}, domain.ErrDeliveryNotFound
}

func (notificationRepoStub) GetByNotificationAndChannel(context.Context, string, domain.Channel) (domain.Delivery, error) {
	return domain.Delivery{}, domain.ErrDeliveryNotFound
}

func (notificationRepoStub) ListByNotification(context.Context, string) ([]domain.Delivery, error) {
	return nil, nil
}

func (notificationRepoStub) ClaimDue(context.Context, string, int) ([]domain.Delivery, error) {
	return nil, nil
}

func (notificationRepoStub) MarkSent(context.Context, string, string, string, int, bool, *time.Time) error {
	return nil
}

func (notificationRepoStub) MarkRetry(context.Context, string, int, time.Time, error) error {
	return nil
}
func (notificationRepoStub) MarkFailed(context.Context, string, int, error) error { return nil }
func (notificationRepoStub) ReleaseLease(context.Context, string, string) error   { return nil }
func (notificationRepoStub) UpdateDeliveryStatus(context.Context, string, domain.DeliveryStatus) error {
	return nil
}

type failingRepo struct {
	notificationRepoStub
	err error
}

func (f failingRepo) Save(context.Context, domain.Notification) error { return f.err }

type countingMetrics struct {
	metrics.Noop
	sent        []string
	suppressed  []string
	prefsUpdate []string
}

func (c *countingMetrics) IncNotificationSent(status string) { c.sent = append(c.sent, status) }

func (c *countingMetrics) IncNotificationSuppressed(notificationType string) {
	c.suppressed = append(c.suppressed, notificationType)
}

func (c *countingMetrics) IncNotificationPreferencesUpdated(status string) {
	c.prefsUpdate = append(c.prefsUpdate, status)
}

func TestSendOrderConfirmation(t *testing.T) {
	tests := []struct {
		name       string
		orderID    string
		userID     string
		totalCents int64
		wantErr    error
	}{
		{"valid", "o-1", "u-1", 1250, nil},
		{"empty order id", "", "u-1", 1250, domain.ErrEmptyOrderID},
		{"empty user id", "o-1", "", 1250, domain.ErrEmptyUserID},
		{"zero total", "o-1", "u-1", 0, domain.ErrNonPositiveTotal},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &countingMetrics{}
			repo := memory.New()
			sender := newFakeEmailSender()
			resolver := newFakeRecipientResolver()
			resolver.SetEmail("u-1", "user@example.com")
			svc := NewService(repo, repo, repo, sender, nil, nil, resolver, nil, m, RetryConfig{})
			err := svc.SendOrderConfirmation(t.Context(), tt.orderID, tt.userID, tt.totalCents)
			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
				assert.Equal(t, []string{metrics.StatusError}, m.sent)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, []string{metrics.StatusSuccess}, m.sent)

			list, err := svc.repo.ListByUserID(t.Context(), tt.userID, 1, 20)
			require.NoError(t, err)
			require.Len(t, list, 1)
			assert.Equal(t, tt.orderID, list[0].OrderID)
			assert.Equal(t, domain.TypeOrderConfirmation, list[0].Type)
		})
	}
}

func TestSendOrderConfirmationPersistenceFailure(t *testing.T) {
	boom := errors.New("disk on fire")
	m := &countingMetrics{}
	sender := newFakeEmailSender()
	resolver := newFakeRecipientResolver()
	resolver.SetEmail("u-1", "user@example.com")
	svc := NewService(failingRepo{err: boom}, notificationRepoStub{}, notificationRepoStub{}, sender, nil, nil, resolver, nil, m, RetryConfig{})

	err := svc.SendOrderConfirmation(t.Context(), "o-1", "u-1", 1250)
	require.ErrorIs(t, err, boom)
	assert.Equal(t, []string{metrics.StatusError}, m.sent)
}

func TestSendOrderConfirmationIsIdempotent(t *testing.T) {
	m := &countingMetrics{}
	repo := memory.New()
	sender := newFakeEmailSender()
	resolver := newFakeRecipientResolver()
	resolver.SetEmail("u-1", "user@example.com")
	svc := NewService(repo, repo, repo, sender, nil, nil, resolver, nil, m, RetryConfig{})

	require.NoError(t, svc.SendOrderConfirmation(t.Context(), "o-1", "u-1", 1250))
	require.NoError(t, svc.SendOrderConfirmation(t.Context(), "o-1", "u-1", 1250))

	list, err := svc.repo.ListByUserID(t.Context(), "u-1", 1, 20)
	require.NoError(t, err)
	assert.Len(t, list, 1, "a redelivered event must not produce a second notification")
}
