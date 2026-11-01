package event

import (
	"context"
	"minimart/internal/notification/application"
	"minimart/internal/notification/domain"
	"minimart/internal/notification/infrastructure/memory"
	"minimart/pkg/metrics"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type countingMetrics struct {
	metrics.Noop
	sent []string
}

func (c *countingMetrics) IncNotificationSent(status string) { c.sent = append(c.sent, status) }

type fakeEmailSender struct {
	mu   sync.Mutex
	sent []domain.Email
	err  error
}

func newFakeEmailSender() *fakeEmailSender {
	return &fakeEmailSender{sent: make([]domain.Email, 0)}
}

func (f *fakeEmailSender) Send(ctx context.Context, e domain.Email) (domain.Receipt, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.sent = append(f.sent, e)
	if f.err != nil {
		return domain.Receipt{}, f.err
	}
	return domain.Receipt{ProviderMessageID: "fake", Delivered: false}, nil
}

func (f *fakeEmailSender) SetErr(err error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.err = err
}

type fakeRecipientResolver struct {
	mu     sync.Mutex
	emails map[string]string
	err    error
}

func newFakeRecipientResolver() *fakeRecipientResolver {
	return &fakeRecipientResolver{emails: make(map[string]string)}
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

func (f *fakeRecipientResolver) SetErr(err error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.err = err
}

func newTestService(m metrics.Metrics) *application.Service {
	repo := memory.New()
	sender := newFakeEmailSender()
	resolver := newFakeRecipientResolver()
	resolver.SetEmail("u1", "user@example.com")
	return application.NewService(repo, repo, repo, sender, nil, nil, resolver, nil, m, application.RetryConfig{})
}

func TestOrderPlaced(t *testing.T) {
	t.Run("malformed JSON returns an error and sends nothing", func(t *testing.T) {
		m := &countingMetrics{}
		h := NewHandler(newTestService(m))

		require.Error(t, h.OrderPlaced(t.Context(), []byte("not json")))
		assert.Empty(t, m.sent)
	})

	t.Run("valid payload triggers a successful confirmation", func(t *testing.T) {
		m := &countingMetrics{}
		h := NewHandler(newTestService(m))

		msg := []byte(`{"order_id":"o1","user_id":"u1","total_cents":1250,"item_count":2}`)
		require.NoError(t, h.OrderPlaced(t.Context(), msg))
		assert.Equal(t, []string{metrics.StatusSuccess}, m.sent)
	})
}
