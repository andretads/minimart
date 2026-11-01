//go:build integration

package test

import (
	"context"
	"minimart/internal/notification/domain"
	mongorepo "minimart/internal/notification/infrastructure/mongo"
	"sync"
	"testing"

	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	"github.com/stretchr/testify/require"
)

type fakeEmailSender struct {
	mu   sync.Mutex
	sent []domain.Email
}

func newFakeEmailSender() *fakeEmailSender {
	return &fakeEmailSender{sent: make([]domain.Email, 0)}
}

func (f *fakeEmailSender) Send(ctx context.Context, e domain.Email) (domain.Receipt, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.sent = append(f.sent, e)
	return domain.Receipt{ProviderMessageID: "fake", Delivered: false}, nil
}

type fakeRecipientResolver struct {
	mu     sync.Mutex
	emails map[string]string
}

func newFakeRecipientResolver() *fakeRecipientResolver {
	return &fakeRecipientResolver{emails: make(map[string]string)}
}

func (f *fakeRecipientResolver) EmailForUser(ctx context.Context, userID string) (string, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if email, ok := f.emails[userID]; ok {
		return email, nil
	}
	return "", domain.ErrRecipientNotFound
}

func (f *fakeRecipientResolver) PhoneForUser(ctx context.Context, userID string) (string, error) {
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

func newTestRepo(t *testing.T) *mongorepo.NotificationRepository {
	t.Helper()
	client, err := mongo.Connect(options.Client().ApplyURI(testMongoURI))
	require.NoError(t, err)
	t.Cleanup(func() { _ = client.Disconnect(t.Context()) })

	repo := mongorepo.NewNotificationRepository(client.Database("minimart"))
	require.NoError(t, repo.EnsureIndexes(t.Context()))
	return repo
}

func newTestPrefsRepo(t *testing.T) *mongorepo.PreferencesRepository {
	t.Helper()
	client, err := mongo.Connect(options.Client().ApplyURI(testMongoURI))
	require.NoError(t, err)
	t.Cleanup(func() { _ = client.Disconnect(t.Context()) })
	return mongorepo.NewPreferencesRepository(client.Database("minimart"))
}

func newTestDeliveryRepo(t *testing.T) *mongorepo.DeliveryRepository {
	t.Helper()
	client, err := mongo.Connect(options.Client().ApplyURI(testMongoURI))
	require.NoError(t, err)
	t.Cleanup(func() { _ = client.Disconnect(t.Context()) })
	repo := mongorepo.NewDeliveryRepository(client.Database("minimart"))
	require.NoError(t, repo.EnsureIndexes(t.Context()))
	return repo
}

func mustNotification(t *testing.T, id, orderID, userID string) domain.Notification {
	t.Helper()
	n, err := domain.NewOrderConfirmation(id, orderID, userID, 1250)
	require.NoError(t, err)
	return n
}

func uid(t *testing.T, suffix string) string {
	t.Helper()
	return t.Name() + "-" + suffix
}
