package application

import (
	"context"
	"minimart/internal/notification/domain"
	"minimart/internal/notification/infrastructure/memory"
	"minimart/pkg/kafka"
	"minimart/pkg/metrics"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakePublisher struct {
	messages []kafka.Message
}

func (f *fakePublisher) Publish(ctx context.Context, msg kafka.Message) error {
	f.messages = append(f.messages, msg)
	return nil
}

func TestDispatchDueDeliveries(t *testing.T) {
	repo := memory.New()
	prefs, _ := domain.NewPreferences("u-1", true, false, false, false)
	require.NoError(t, repo.Upsert(t.Context(), prefs))
	resolver := newFakeRecipientResolver()
	resolver.SetEmail("u-1", "user@example.com")
	publisher := &fakePublisher{}
	svc := NewService(repo, repo, repo, newFakeEmailSender(), nil, nil, resolver, publisher, metrics.Noop{}, RetryConfig{MaxAttempts: 3, BaseDelay: time.Millisecond, MaxDelay: time.Second})

	n, err := domain.NewOrderConfirmation(newID(), "o-1", "u-1", 1250)
	require.NoError(t, err)
	_, deliveries, err := svc.IngestNotification(t.Context(), n, nil)
	require.NoError(t, err)
	require.Len(t, deliveries, 1)

	published, err := svc.DispatchDueDeliveries(t.Context(), "worker-1", 10)
	require.NoError(t, err)
	assert.Equal(t, 1, published)
	require.Len(t, publisher.messages, 1)
	payload, ok := publisher.messages[0].Payload.([]byte)
	require.True(t, ok)
	assert.Equal(t, deliveries[0].ID, string(payload))
}
