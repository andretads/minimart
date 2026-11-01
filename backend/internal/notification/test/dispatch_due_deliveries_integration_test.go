//go:build integration

package test

import (
	"context"
	"minimart/internal/notification/application"
	"minimart/pkg/kafka"
	"minimart/pkg/metrics"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type intFakePublisher struct {
	messages []kafka.Message
}

func (f *intFakePublisher) Publish(ctx context.Context, msg kafka.Message) error {
	f.messages = append(f.messages, msg)
	return nil
}

func TestDispatchDueDeliveriesIntegration(t *testing.T) {
	ctx := t.Context()
	repo := newTestRepo(t)
	deliveryRepo := newTestDeliveryRepo(t)
	prefsRepo := newTestPrefsRepo(t)
	resolver := newFakeRecipientResolver()
	pub := &intFakePublisher{}
	svc := application.NewService(repo, deliveryRepo, prefsRepo, newFakeEmailSender(), nil, nil, resolver, pub, metrics.Noop{}, application.RetryConfig{})

	t.Run("No due deliveries publishes nothing", func(t *testing.T) {
		published, err := svc.DispatchDueDeliveries(ctx, "w1", 10)
		require.NoError(t, err)
		assert.Equal(t, 0, published)
	})
}
