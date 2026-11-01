package kafka

import (
	"context"
	"encoding/json"
	"errors"
	"minimart/pkg/metrics"
	"testing"
	"time"

	wmessage "github.com/ThreeDotsLabs/watermill/message"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/goleak"
)

func TestMain(m *testing.M) {
	goleak.VerifyTestMain(m)
}

type publishedMessage struct {
	msg Message
	err error
}

// fakeHeaderPublisher records published messages. fail makes every publish fail.
type fakeHeaderPublisher struct {
	published []publishedMessage
	fail      error
}

func (f *fakeHeaderPublisher) Publish(_ context.Context, msg Message) error {
	f.published = append(f.published, publishedMessage{msg: msg, err: f.fail})
	return f.fail
}

func newTestConsumer(dlq HeaderPublisher, policy RetryPolicy) (*Consumer, *[]time.Duration) {
	var delays []time.Duration
	c := NewConsumerWithRetry(nil, dlq, metrics.Noop{}, policy)
	c.jitter = func(d time.Duration) time.Duration { return d }
	c.sleep = func(_ context.Context, d time.Duration) error {
		delays = append(delays, d)
		return nil
	}
	return c, &delays
}

func testMessage(payload string) *wmessage.Message {
	msg := wmessage.NewMessage("message-1", []byte(payload))
	msg.Metadata.Set("key", "aggregate-1")
	return msg
}

func assertAcked(t *testing.T, msg *wmessage.Message) {
	t.Helper()
	require.NotNil(t, msg.Acked())
	select {
	case <-msg.Acked():
	default:
		t.Fatal("message was not acknowledged")
	}
}

func assertNotAcked(t *testing.T, msg *wmessage.Message) {
	t.Helper()
	if msg.Acked() == nil {
		return
	}
	select {
	case <-msg.Acked():
		t.Fatal("message was acknowledged")
	default:
	}
}

func TestHandleMessageSuccessAcksWithoutRetry(t *testing.T) {
	dlq := &fakeHeaderPublisher{}
	c, delays := newTestConsumer(dlq, RetryPolicy{MaxAttempts: 3, BaseDelay: time.Millisecond, MaxDelay: time.Second})
	msg := testMessage(`{"ok":true}`)

	calls := 0
	err := c.handleMessage(t.Context(), OrderPlacedTopic, OrderConfirmationConsumer, msg, func(context.Context, []byte) error {
		calls++
		return nil
	})

	require.NoError(t, err)
	assert.Equal(t, 1, calls)
	assert.Empty(t, *delays)
	assert.Empty(t, dlq.published)
	assertAcked(t, msg)
}

func TestHandleMessageRetriesThenSucceeds(t *testing.T) {
	dlq := &fakeHeaderPublisher{}
	c, delays := newTestConsumer(dlq, RetryPolicy{MaxAttempts: 4, BaseDelay: 100 * time.Millisecond, MaxDelay: time.Second})
	msg := testMessage(`{"ok":true}`)

	calls := 0
	err := c.handleMessage(t.Context(), OrderPlacedTopic, OrderConfirmationConsumer, msg, func(context.Context, []byte) error {
		calls++
		if calls < 3 {
			return errors.New("transient")
		}
		return nil
	})

	require.NoError(t, err)
	assert.Equal(t, 3, calls)
	assert.Equal(t, []time.Duration{100 * time.Millisecond, 200 * time.Millisecond}, *delays)
	assert.Empty(t, dlq.published)
	assertAcked(t, msg)
}

func TestHandleMessageExhaustedBudgetDeadLetters(t *testing.T) {
	dlq := &fakeHeaderPublisher{}
	c, delays := newTestConsumer(dlq, RetryPolicy{MaxAttempts: 3, BaseDelay: 10 * time.Millisecond, MaxDelay: time.Second})
	msg := testMessage(`{"ok":true}`)
	msg.Metadata.Set("x-minimart-stale", "old")

	calls := 0
	err := c.handleMessage(t.Context(), OrderPlacedTopic, OrderConfirmationConsumer, msg, func(context.Context, []byte) error {
		calls++
		return errors.New("still failing")
	})

	require.NoError(t, err)
	assert.Equal(t, 3, calls)
	assert.Len(t, *delays, 2)
	require.Len(t, dlq.published, 1)

	got := dlq.published[0].msg
	assert.Equal(t, DLQTopic(OrderPlacedTopic), got.Topic)
	assert.Equal(t, "aggregate-1", got.Key)
	assert.Equal(t, msg.Payload, got.Payload)
	assert.Equal(t, OrderPlacedTopic.String(), got.Headers[DLQHeaderOriginalTopic])
	assert.Equal(t, OrderConfirmationConsumer, got.Headers[DLQHeaderConsumerGroup])
	assert.Equal(t, "3", got.Headers[DLQHeaderAttempts])
	assert.Equal(t, DLQDispositionExhausted, got.Headers[DLQHeaderDisposition])
	assert.Equal(t, dlqVersion, got.Headers[DLQHeaderVersion])
	assert.Contains(t, got.Headers[DLQHeaderError], "still failing")
	assert.NotEmpty(t, got.Headers[DLQHeaderFailedAt])
	assert.NotEmpty(t, got.Headers[DLQHeaderFirstFailedAt])
	_, stale := got.Headers["x-minimart-stale"]
	assert.False(t, stale)
	assertAcked(t, msg)
}

func TestHandleMessagePermanentErrorDeadLettersImmediately(t *testing.T) {
	dlq := &fakeHeaderPublisher{}
	c, delays := newTestConsumer(dlq, RetryPolicy{MaxAttempts: 5, BaseDelay: time.Millisecond, MaxDelay: time.Second})
	msg := testMessage(`{"ok":true}`)

	calls := 0
	err := c.handleMessage(t.Context(), ReviewCreatedTopic, ReviewModerationConsumer, msg, func(context.Context, []byte) error {
		calls++
		return Permanent(errors.New("moderation rejected"))
	})

	require.NoError(t, err)
	assert.Equal(t, 1, calls)
	assert.Empty(t, *delays)
	require.Len(t, dlq.published, 1)
	assert.Equal(t, DLQDispositionPermanent, dlq.published[0].msg.Headers[DLQHeaderDisposition])
	assertAcked(t, msg)
}

func TestHandleMessageMalformedJSONIsPermanent(t *testing.T) {
	dlq := &fakeHeaderPublisher{}
	c, delays := newTestConsumer(dlq, RetryPolicy{MaxAttempts: 5, BaseDelay: time.Millisecond, MaxDelay: time.Second})
	msg := testMessage(`not-json`)

	calls := 0
	err := c.handleMessage(t.Context(), OrderPlacedTopic, OrderConfirmationConsumer, msg, func(_ context.Context, body []byte) error {
		calls++
		var v struct {
			ID int `json:"id"`
		}
		return json.Unmarshal(body, &v)
	})

	require.NoError(t, err)
	assert.Equal(t, 1, calls)
	assert.Empty(t, *delays)
	require.Len(t, dlq.published, 1)
	assert.Equal(t, DLQDispositionPermanent, dlq.published[0].msg.Headers[DLQHeaderDisposition])
	assertAcked(t, msg)
}

func TestHandleMessageCancellationDoesNotAckOrDeadLetter(t *testing.T) {
	dlq := &fakeHeaderPublisher{}
	c, _ := newTestConsumer(dlq, RetryPolicy{MaxAttempts: 5, BaseDelay: time.Millisecond, MaxDelay: time.Second})
	msg := testMessage(`{"ok":true}`)

	ctx, cancel := context.WithCancel(t.Context())
	err := c.handleMessage(ctx, OrderPlacedTopic, OrderConfirmationConsumer, msg, func(context.Context, []byte) error {
		cancel()
		return errors.New("interrupted")
	})

	require.ErrorIs(t, err, context.Canceled)
	assert.Empty(t, dlq.published)
	assertNotAcked(t, msg)
}

func TestHandleMessageDLQPublishFailureLeavesMessageUnacked(t *testing.T) {
	dlq := &fakeHeaderPublisher{fail: errors.New("broker down")}
	c, _ := newTestConsumer(dlq, RetryPolicy{MaxAttempts: 2, BaseDelay: time.Millisecond, MaxDelay: time.Second})
	msg := testMessage(`{"ok":true}`)

	err := c.handleMessage(t.Context(), OrderPlacedTopic, OrderConfirmationConsumer, msg, func(context.Context, []byte) error {
		return Permanent(errors.New("bad data"))
	})

	require.Error(t, err)
	require.Len(t, dlq.published, 1)
	assertNotAcked(t, msg)
}

func TestHandleMessageWithoutDLQPublishLeavesMessageUnacked(t *testing.T) {
	c, _ := newTestConsumer(nil, RetryPolicy{MaxAttempts: 2, MaxDelay: time.Second})
	msg := testMessage(`{"ok":true}`)

	err := c.handleMessage(t.Context(), OrderPlacedTopic, OrderConfirmationConsumer, msg, func(context.Context, []byte) error {
		return errors.New("transient")
	})

	require.Error(t, err)
	assertNotAcked(t, msg)
}

func TestBackoffDoublesAndCaps(t *testing.T) {
	c, _ := newTestConsumer(nil, RetryPolicy{MaxAttempts: 6, BaseDelay: 100 * time.Millisecond, MaxDelay: 500 * time.Millisecond})

	assert.Equal(t, 100*time.Millisecond, c.backoff(1))
	assert.Equal(t, 200*time.Millisecond, c.backoff(2))
	assert.Equal(t, 400*time.Millisecond, c.backoff(3))
	assert.Equal(t, 500*time.Millisecond, c.backoff(4))
	assert.Equal(t, 500*time.Millisecond, c.backoff(10))
}
