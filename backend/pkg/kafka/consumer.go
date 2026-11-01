package kafka

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math/rand/v2"
	"minimart/pkg/backpressure"
	"minimart/pkg/metrics"
	"strconv"
	"strings"
	"time"

	"github.com/IBM/sarama"
	"github.com/ThreeDotsLabs/watermill"
	wkafka "github.com/ThreeDotsLabs/watermill-kafka/v3/pkg/kafka"
	wmessage "github.com/ThreeDotsLabs/watermill/message"
)

// RetryPolicy bounds how many times a handler is attempted and how long the
// consumer waits between attempts. A message is only acknowledged after it
// succeeds or is safely stored in the dead-letter topic.
type RetryPolicy struct {
	// MaxAttempts is the total number of handler calls per message. Values below
	// one are normalized to one (no retries).
	MaxAttempts int
	// BaseDelay is the first backoff. It doubles on each retry.
	BaseDelay time.Duration
	// MaxDelay caps the backoff before jitter is applied.
	MaxDelay time.Duration
}

// DefaultRetryPolicy is used by NewConsumer.
func DefaultRetryPolicy() RetryPolicy {
	return RetryPolicy{
		MaxAttempts: 5,
		BaseDelay:   500 * time.Millisecond,
		MaxDelay:    30 * time.Second,
	}
}

func (p RetryPolicy) normalized() RetryPolicy {
	if p.MaxAttempts < 1 {
		p.MaxAttempts = 1
	}
	if p.BaseDelay < 0 {
		p.BaseDelay = 0
	}
	if p.MaxDelay < p.BaseDelay {
		p.MaxDelay = p.BaseDelay
	}
	return p
}

// ConsumerOptions configures optional consumer behavior.
type ConsumerOptions struct {
	// Gate pauses consumption when dependencies are saturated.
	Gate *backpressure.Gate
	// Auth carries optional Kafka authentication settings.
	Auth Auth
}

// Consumer subscribes to topics and dispatches raw message bodies to handlers.
type Consumer struct {
	brokers []string
	dlq     HeaderPublisher // optional; failed messages are routed here
	logger  watermill.LoggerAdapter
	m       metrics.Metrics
	retry   RetryPolicy
	gate    *backpressure.Gate
	auth    Auth

	// Injected in tests to avoid real waiting and non-deterministic jitter.
	sleep  func(context.Context, time.Duration) error
	jitter func(time.Duration) time.Duration
}

func NewConsumer(brokers []string, dlq HeaderPublisher, m metrics.Metrics) *Consumer {
	return NewConsumerWithRetry(brokers, dlq, m, DefaultRetryPolicy())
}

// NewConsumerWithRetry builds a consumer with an explicit retry policy.
func NewConsumerWithRetry(brokers []string, dlq HeaderPublisher, m metrics.Metrics, retry RetryPolicy) *Consumer {
	return NewConsumerWithOptions(brokers, dlq, m, retry, ConsumerOptions{})
}

// NewConsumerWithOptions builds a consumer with an explicit retry policy and options.
func NewConsumerWithOptions(brokers []string, dlq HeaderPublisher, m metrics.Metrics, retry RetryPolicy, opts ConsumerOptions) *Consumer {
	if m == nil {
		m = metrics.Noop{}
	}
	return &Consumer{
		brokers: brokers,
		dlq:     dlq,
		logger:  watermillLogAdapter(),
		m:       m,
		retry:   retry.normalized(),
		gate:    opts.Gate,
		auth:    opts.Auth,
		sleep:   sleepContext,
		jitter:  fullJitter,
	}
}

// Subscribe joins consumerGroup on topic and calls handler for each message body.
// It blocks until ctx is cancelled. The handler receives that same ctx as its
// first argument so downstream work is cancelled with the consumer, mirroring an
// HTTP handler taking the request context.
//
// A handler error is retried with exponential backoff. A permanent error (see
// Permanent) or an exhausted retry budget routes the message to a dead-letter
// topic (<topic>.dlq) and only then acknowledges it. If the DLQ publish fails, or
// the consumer is shutting down, the original message is left unacknowledged so
// Kafka redelivers it; one poison message cannot block the partition forever
// without a durable record of the failure.
func (c *Consumer) Subscribe(ctx context.Context, topic Topic, consumerGroup string, handler func(ctx context.Context, msg []byte) error) error {
	cfg := wkafka.DefaultSaramaSubscriberConfig()
	// Same version pin as the producer - see producer.go.
	cfg.Version = sarama.V2_1_0_0
	cfg.Consumer.Offsets.Initial = sarama.OffsetOldest
	applyAuth(cfg, c.auth)

	sub, err := wkafka.NewSubscriber(
		wkafka.SubscriberConfig{
			Brokers:               c.brokers,
			Unmarshaler:           wkafka.DefaultMarshaler{},
			OverwriteSaramaConfig: cfg,
			ConsumerGroup:         consumerGroup,
		},
		c.logger,
	)
	if err != nil {
		return fmt.Errorf("new kafka subscriber: %w", err)
	}
	defer func() { _ = sub.Close() }()

	messages, err := sub.Subscribe(ctx, topic.String())
	if err != nil {
		return fmt.Errorf("subscribe %s: %w", topic, err)
	}

	for {
		if c.gate != nil {
			if c.gate.Paused() {
				c.m.IncKafkaBackpressureWait(topic.String(), consumerGroup)
				c.m.SetKafkaConsumerPaused(topic.String(), consumerGroup, true)
			}
			if err := c.gate.Wait(ctx); err != nil {
				return nil
			}
			c.m.SetKafkaConsumerPaused(topic.String(), consumerGroup, c.gate.Paused())
		}
		select {
		case <-ctx.Done():
			return nil
		case msg, ok := <-messages:
			if !ok {
				if ctx.Err() != nil {
					return nil
				}
				return fmt.Errorf("subscription for %s closed unexpectedly", topic)
			}
			if err := c.handleMessage(ctx, topic, consumerGroup, msg, handler); err != nil {
				if ctx.Err() != nil {
					return nil
				}
				return err
			}
		}
	}
}

// handleMessage runs the retry loop for one message. It returns an error only
// when the subscription should stop (the DLQ publish failed or no DLQ publisher
// is configured); a cancelled context is reported through ctx.Err().
func (c *Consumer) handleMessage(ctx context.Context, topic Topic, group string, msg *wmessage.Message, handler func(context.Context, []byte) error) error {
	start := time.Now()
	var (
		attempts      int
		firstFailedAt time.Time
		lastErr       error
	)

	for attempts < c.retry.MaxAttempts {
		if err := ctx.Err(); err != nil {
			return err
		}
		attempts++

		err := handler(ctx, msg.Payload)
		if err == nil {
			c.recordConsumed(topic, group, metrics.StatusSuccess, start)
			msg.Ack()
			return nil
		}
		lastErr = err
		if firstFailedAt.IsZero() {
			firstFailedAt = time.Now().UTC()
		}

		if isPermanent(err) || attempts >= c.retry.MaxAttempts {
			break
		}

		delay := c.backoff(attempts)
		c.logger.Info("kafka handler failed; retrying", watermill.LogFields{
			"topic":        topic.String(),
			"group":        group,
			"attempt":      attempts,
			"max_attempts": c.retry.MaxAttempts,
			"delay":        delay.String(),
			"error":        err.Error(),
		})
		if err := c.sleep(ctx, delay); err != nil {
			return err
		}
		c.m.IncKafkaRetried(topic.String(), group)
	}

	if err := ctx.Err(); err != nil {
		return err
	}
	if err := c.deadLetter(ctx, topic, group, msg, attempts, firstFailedAt, lastErr); err != nil {
		return err
	}
	c.recordConsumed(topic, group, metrics.StatusError, start)
	msg.Ack()
	return nil
}

func (c *Consumer) deadLetter(ctx context.Context, topic Topic, group string, msg *wmessage.Message, attempts int, firstFailedAt time.Time, cause error) error {
	fields := watermill.LogFields{
		"topic":       topic.String(),
		"group":       group,
		"attempts":    attempts,
		"dlq_topic":   DLQTopic(topic).String(),
		"message_key": msg.Metadata.Get("key"),
	}
	if c.dlq == nil {
		c.logger.Error("handler failed and no DLQ publisher is configured; leaving message unacked", cause, fields)
		return fmt.Errorf("handler failed for %s and no DLQ publisher is configured: %w", topic, cause)
	}

	disposition := DLQDispositionExhausted
	if isPermanent(cause) {
		disposition = DLQDispositionPermanent
	}
	headers := dlqHeaders(topic, group, msg, attempts, firstFailedAt, cause, disposition)
	if err := c.dlq.Publish(ctx, Message{
		Topic:   DLQTopic(topic),
		Key:     msg.Metadata.Get("key"),
		Payload: msg.Payload,
		Headers: headers,
	}); err != nil {
		c.logger.Error("failed to publish to DLQ; leaving message unacked", err, fields)
		return fmt.Errorf("publish to %s: %w", DLQTopic(topic), err)
	}

	c.m.IncKafkaDeadLettered(topic.String(), group, disposition)
	c.logger.Error("kafka message dead-lettered", cause, fields.Add(watermill.LogFields{
		"disposition": disposition,
	}))
	return nil
}

func (c *Consumer) recordConsumed(topic Topic, group, status string, start time.Time) {
	c.m.IncKafkaConsumed(topic.String(), group, status)
	c.m.ObserveKafkaConsumeLatency(topic.String(), group, time.Since(start))
}

// backoff returns the delay before retry number attempt+1. attempt starts at 1.
func (c *Consumer) backoff(attempt int) time.Duration {
	d := c.retry.BaseDelay
	for i := 1; i < attempt && d < c.retry.MaxDelay; i++ {
		d *= 2
	}
	if d > c.retry.MaxDelay {
		d = c.retry.MaxDelay
	}
	if d <= 0 {
		return 0
	}
	return c.jitter(d)
}

func fullJitter(d time.Duration) time.Duration {
	if d <= 0 {
		return 0
	}
	//nolint:gosec // retry jitter only spreads load; it does not need cryptographic randomness
	return time.Duration(rand.Int64N(int64(d) + 1))
}

func sleepContext(ctx context.Context, d time.Duration) error {
	if d <= 0 {
		return ctx.Err()
	}
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-t.C:
		return nil
	}
}

// isPermanent covers errors that cannot become valid data on retry: an explicit
// Permanent marker and JSON decoding failures.
func isPermanent(err error) bool {
	if IsPermanent(err) {
		return true
	}
	if _, ok := errors.AsType[*json.SyntaxError](err); ok {
		return true
	}
	_, ok := errors.AsType[*json.UnmarshalTypeError](err)
	return ok
}

const (
	minimartHeaderPrefix = "x-minimart-"

	// DLQHeaderOriginalTopic identifies the topic the message failed on.
	DLQHeaderOriginalTopic = minimartHeaderPrefix + "original-topic"
	// DLQHeaderOriginalPartition is the source partition, when available.
	DLQHeaderOriginalPartition = minimartHeaderPrefix + "original-partition"
	// DLQHeaderOriginalOffset is the source offset, when available.
	DLQHeaderOriginalOffset = minimartHeaderPrefix + "original-offset"
	// DLQHeaderConsumerGroup is the consumer group that failed the message.
	DLQHeaderConsumerGroup = minimartHeaderPrefix + "consumer-group"
	// DLQHeaderAttempts is the number of handler calls made.
	DLQHeaderAttempts = minimartHeaderPrefix + "attempts"
	// DLQHeaderDisposition is "permanent" or "exhausted".
	DLQHeaderDisposition = minimartHeaderPrefix + "disposition"
	// DLQHeaderFirstFailedAt is when the first attempt failed (RFC 3339 nano).
	DLQHeaderFirstFailedAt = minimartHeaderPrefix + "first-failed-at"
	// DLQHeaderFailedAt is when the message was dead-lettered (RFC 3339 nano).
	DLQHeaderFailedAt = minimartHeaderPrefix + "failed-at"
	// DLQHeaderError is the truncated final handler error.
	DLQHeaderError = minimartHeaderPrefix + "error"
	// DLQHeaderVersion versions the dead-letter metadata contract.
	DLQHeaderVersion = minimartHeaderPrefix + "dlq-version"
	// DLQHeaderReplayCount tracks how many times a message has been replayed.
	DLQHeaderReplayCount = minimartHeaderPrefix + "replay-count"

	DLQDispositionPermanent = "permanent"
	DLQDispositionExhausted = "exhausted"

	dlqVersion       = "1"
	maxDLErrorLength = 1024
)

func dlqHeaders(topic Topic, group string, msg *wmessage.Message, attempts int, firstFailedAt time.Time, cause error, disposition string) map[string]string {
	headers := make(map[string]string, len(msg.Metadata)+8)
	for k, v := range msg.Metadata {
		if strings.HasPrefix(k, minimartHeaderPrefix) {
			continue
		}
		headers[k] = v
	}

	headers[DLQHeaderOriginalTopic] = topic.String()
	headers[DLQHeaderConsumerGroup] = group
	headers[DLQHeaderAttempts] = strconv.Itoa(attempts)
	headers[DLQHeaderDisposition] = disposition
	headers[DLQHeaderFailedAt] = time.Now().UTC().Format(time.RFC3339Nano)
	headers[DLQHeaderError] = truncate(cause.Error(), maxDLErrorLength)
	headers[DLQHeaderVersion] = dlqVersion
	if !firstFailedAt.IsZero() {
		headers[DLQHeaderFirstFailedAt] = firstFailedAt.Format(time.RFC3339Nano)
	}

	// Watermill publishes partition/offset/timestamp on the message context. They
	// are optional: tests and producers built by hand may not set them.
	msgCtx := msg.Context()
	if partition, ok := wkafka.MessagePartitionFromCtx(msgCtx); ok {
		headers[DLQHeaderOriginalPartition] = strconv.Itoa(int(partition))
	}
	if offset, ok := wkafka.MessagePartitionOffsetFromCtx(msgCtx); ok {
		headers[DLQHeaderOriginalOffset] = strconv.FormatInt(offset, 10)
	}
	return headers
}

func truncate(s string, limit int) string {
	if len(s) <= limit {
		return s
	}
	return s[:limit]
}
