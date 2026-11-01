package application

import (
	"context"
	"errors"
	"minimart/internal/notification/domain"
	"minimart/internal/notification/infrastructure/memory"
	"minimart/pkg/kafka"
	"minimart/pkg/metrics"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakePublisher struct {
	messages []kafka.Message
	err      error
}

func (f *fakePublisher) Publish(ctx context.Context, msg kafka.Message) error {
	if f.err != nil {
		return f.err
	}
	f.messages = append(f.messages, msg)
	return nil
}

func TestDispatchDueDeliveries(t *testing.T) {
	t.Run("No due deliveries publishes nothing", func(t *testing.T) {
		repo := memory.New()
		svc := NewService(repo, repo, repo, newFakeEmailSender(), nil, nil, newFakeRecipientResolver(), &fakePublisher{}, metrics.Noop{}, RetryConfig{})
		published, err := svc.DispatchDueDeliveries(t.Context(), "w1", 10)
		require.NoError(t, err)
		assert.Equal(t, 0, published)
	})

	t.Run("A due delivery is claimed and published to its channel topic", func(t *testing.T) {
		repo := memory.New()
		prefs, _ := domain.NewPreferences("u1", true, false, false, false)
		require.NoError(t, repo.Upsert(t.Context(), prefs))
		resolver := newFakeRecipientResolver()
		resolver.SetEmail("u1", "user@example.com")
		publisher := &fakePublisher{}
		svc := NewService(repo, repo, repo, newFakeEmailSender(), nil, nil, resolver, publisher, metrics.Noop{}, RetryConfig{})
		n, err := domain.NewOrderConfirmation(newID(), "o1", "u1", 1250)
		require.NoError(t, err)
		_, deliveries, err := svc.IngestNotification(t.Context(), n, nil)
		require.NoError(t, err)
		require.Len(t, deliveries, 1)

		published, err := svc.DispatchDueDeliveries(t.Context(), "w1", 10)
		require.NoError(t, err)
		assert.Equal(t, 1, published)
		require.Len(t, publisher.messages, 1)
		payload, ok := publisher.messages[0].Payload.([]byte)
		require.True(t, ok)
		assert.Equal(t, deliveries[0].ID, string(payload))
		assert.Equal(t, kafka.Topic("notification-email"), publisher.messages[0].Topic)
	})

	t.Run("A publish failure releases the lease and is reported", func(t *testing.T) {
		repo := memory.New()
		prefs, _ := domain.NewPreferences("u1", true, false, false, false)
		require.NoError(t, repo.Upsert(t.Context(), prefs))
		resolver := newFakeRecipientResolver()
		resolver.SetEmail("u1", "user@example.com")
		publisher := &fakePublisher{err: errors.New("publish delivery")}
		svc := NewService(repo, repo, repo, newFakeEmailSender(), nil, nil, resolver, publisher, metrics.Noop{}, RetryConfig{})
		n, err := domain.NewOrderConfirmation(newID(), "o1", "u1", 1250)
		require.NoError(t, err)
		_, deliveries, err := svc.IngestNotification(t.Context(), n, nil)
		require.NoError(t, err)
		require.Len(t, deliveries, 1)

		published, err := svc.DispatchDueDeliveries(t.Context(), "w1", 10)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "publish delivery")
		assert.Equal(t, 0, published)
		d, err := repo.GetDeliveryByID(t.Context(), deliveries[0].ID)
		require.NoError(t, err)
		assert.Empty(t, d.LockedBy)
	})

	t.Run("A default batch size is used when none is requested", func(t *testing.T) {
		repo := memory.New()
		prefs, _ := domain.NewPreferences("u1", true, false, false, false)
		require.NoError(t, repo.Upsert(t.Context(), prefs))
		resolver := newFakeRecipientResolver()
		resolver.SetEmail("u1", "user@example.com")
		publisher := &fakePublisher{}
		svc := NewService(repo, repo, repo, newFakeEmailSender(), nil, nil, resolver, publisher, metrics.Noop{}, RetryConfig{})
		n, err := domain.NewOrderConfirmation(newID(), "o1", "u1", 1250)
		require.NoError(t, err)
		_, _, err = svc.IngestNotification(t.Context(), n, nil)
		require.NoError(t, err)

		published, err := svc.DispatchDueDeliveries(t.Context(), "w1", 0)
		require.NoError(t, err)
		assert.Equal(t, 1, published)
	})
}
