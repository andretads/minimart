package worker

import (
	"context"
	"errors"
	"minimart/pkg/kafka"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/goleak"
)

func TestMain(m *testing.M) {
	goleak.VerifyTestMain(m)
}

// fakeSubscriber stands in for kafka.Consumer. onSubscribe drives its behaviour.
type fakeSubscriber struct {
	onSubscribe func(ctx context.Context, handler func(ctx context.Context, msg []byte) error) error
}

func (f *fakeSubscriber) Subscribe(ctx context.Context, _ kafka.Topic, _ string, handler func(ctx context.Context, msg []byte) error) error {
	if f.onSubscribe != nil {
		return f.onSubscribe(ctx, handler)
	}
	<-ctx.Done()
	return nil
}

type fakePublisher struct{}

func (fakePublisher) PublishMessage(context.Context, kafka.Topic, string, any) error { return nil }

func TestAttachKafkaRunsUntilContextCancelled(t *testing.T) {
	sub := &fakeSubscriber{} // blocks until ctx is done
	w := New(sub, fakePublisher{})

	ctx, cancel := context.WithCancel(t.Context())
	w.AttachKafka(ctx, kafka.OrderPlacedTopic, kafka.OrderConfirmationConsumer, func(context.Context, []byte) error {
		return nil
	})
	cancel()
	w.Wait() // must return; goleak in TestMain proves the goroutine exited.
}

func TestAttachKafkaRestartsWhenSubscriberErrors(t *testing.T) {
	var calls atomic.Int32
	restarted := make(chan struct{}, 1)
	sub := &fakeSubscriber{onSubscribe: func(context.Context, func(context.Context, []byte) error) error {
		if calls.Add(1) > 1 {
			select {
			case restarted <- struct{}{}:
			default:
			}
		}
		return errors.New("consumer blew up")
	}}
	w := NewWithConfig(sub, fakePublisher{}, Config{
		RestartMinBackoff: time.Millisecond,
		RestartMaxBackoff: time.Millisecond,
	})

	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	w.AttachKafka(ctx, kafka.OrderPlacedTopic, kafka.OrderConfirmationConsumer, func(context.Context, []byte) error {
		return nil
	})

	select {
	case <-restarted:
	case <-time.After(5 * time.Second):
		t.Fatal("consumer was not restarted after an error")
	}
	cancel()
	w.Wait()
	assert.GreaterOrEqual(t, calls.Load(), int32(2))
}

func TestAttachKafkaInvokesHandler(t *testing.T) {
	var got []byte
	handled := make(chan struct{})
	sub := &fakeSubscriber{onSubscribe: func(ctx context.Context, handler func(context.Context, []byte) error) error {
		if err := handler(ctx, []byte("payload")); err != nil {
			return err
		}
		<-ctx.Done()
		return nil
	}}
	w := New(sub, fakePublisher{})

	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	w.AttachKafka(ctx, kafka.OrderPlacedTopic, kafka.OrderConfirmationConsumer, func(_ context.Context, msg []byte) error {
		got = msg
		close(handled)
		return nil
	})

	select {
	case <-handled:
	case <-time.After(5 * time.Second):
		t.Fatal("handler was not invoked")
	}
	cancel()
	w.Wait()
	assert.Equal(t, []byte("payload"), got)
}

func TestProducerExposesInjectedPublisher(t *testing.T) {
	pub := fakePublisher{}
	w := New(&fakeSubscriber{}, pub)
	require.Equal(t, pub, w.Producer())
}

func TestAttachKafkaStartsConfiguredReplicas(t *testing.T) {
	var replicas atomic.Int32
	release := make(chan struct{})
	sub := &fakeSubscriber{onSubscribe: func(ctx context.Context, _ func(ctx context.Context, msg []byte) error) error {
		replicas.Add(1)
		select {
		case <-release:
		case <-ctx.Done():
		}
		return nil
	}}
	w := NewWithConfig(sub, fakePublisher{}, Config{KafkaConcurrency: 3})

	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	w.AttachKafka(ctx, kafka.OrderPlacedTopic, kafka.OrderConfirmationConsumer, func(context.Context, []byte) error {
		return nil
	})

	assert.Eventually(t, func() bool { return replicas.Load() == 3 }, 5*time.Second, 10*time.Millisecond)
	close(release)
	cancel()
	w.Wait()
}

func TestSubscriptionsTrackEveryTopicAndGroupPair(t *testing.T) {
	w := New(&fakeSubscriber{}, fakePublisher{})

	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	w.AttachKafka(ctx, kafka.PaymentSucceededTopic, kafka.OrderPaymentConsumer, func(context.Context, []byte) error {
		return nil
	})
	w.AttachKafka(ctx, kafka.PaymentSucceededTopic, kafka.NotificationConsumer, func(context.Context, []byte) error {
		return nil
	})

	assert.Equal(t, []kafka.Subscription{
		{Topic: kafka.PaymentSucceededTopic, Group: kafka.NotificationConsumer},
		{Topic: kafka.PaymentSucceededTopic, Group: kafka.OrderPaymentConsumer},
	}, w.Subscriptions())
}
