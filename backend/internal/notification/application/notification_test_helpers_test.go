package application

import (
	"context"
	"minimart/internal/notification/domain"
	"minimart/pkg/metrics"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

type fakeEmailSender struct {
	mu      sync.Mutex
	sent    []domain.Email
	sendErr error
}

func newFakeEmailSender() *fakeEmailSender {
	return &fakeEmailSender{sent: make([]domain.Email, 0)}
}

func (f *fakeEmailSender) Send(ctx context.Context, e domain.Email) (domain.Receipt, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.sent = append(f.sent, e)
	if f.sendErr != nil {
		return domain.Receipt{}, f.sendErr
	}
	return domain.Receipt{ProviderMessageID: "fake", Delivered: false}, nil
}

func (f *fakeEmailSender) SetSendErr(err error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.sendErr = err
}

func (f *fakeEmailSender) GetSent() []domain.Email {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]domain.Email(nil), f.sent...)
}

type fakeRecipientResolver struct {
	mu     sync.Mutex
	emails map[string]string
	phones map[string]string
	err    error
}

func newFakeRecipientResolver() *fakeRecipientResolver {
	return &fakeRecipientResolver{emails: make(map[string]string), phones: make(map[string]string)}
}

func (f *fakeRecipientResolver) EmailForUser(ctx context.Context, userID string) (string, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.err != nil {
		return "", f.err
	}
	if email, ok := f.emails[userID]; ok {
		return email, nil
	}
	return "", domain.ErrRecipientNotFound
}

func (f *fakeRecipientResolver) PhoneForUser(ctx context.Context, userID string) (string, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.err != nil {
		return "", f.err
	}
	if phone, ok := f.phones[userID]; ok {
		return phone, nil
	}
	return "", domain.ErrRecipientNotFound
}

func (f *fakeRecipientResolver) PushTokenForUser(ctx context.Context, userID string) (string, error) {
	return "", domain.ErrRecipientNotFound
}

func (f *fakeRecipientResolver) SetEmail(userID, email string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.emails[userID] = email
}

func (f *fakeRecipientResolver) SetPhone(userID, phone string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.phones[userID] = phone
}

func (f *fakeRecipientResolver) SetErr(err error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.err = err
}

func mustNotification(t *testing.T, id, orderID, userID string) domain.Notification {
	t.Helper()
	n, err := domain.NewOrderConfirmation(id, orderID, userID, 1000)
	require.NoError(t, err)
	return n
}

type failingPrefsRepo struct{ err error }

func (f failingPrefsRepo) GetByUserID(context.Context, string) (domain.Preferences, error) {
	return domain.Preferences{}, f.err
}

func (f failingPrefsRepo) Upsert(context.Context, domain.Preferences) error { return f.err }

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
