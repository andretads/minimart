package kafka

import (
	"context"
	"encoding/json"
	"fmt"
	"minimart/pkg/metrics"
	"time"

	"github.com/IBM/sarama"
	"github.com/ThreeDotsLabs/watermill"
	wkafka "github.com/ThreeDotsLabs/watermill-kafka/v3/pkg/kafka"
	"github.com/ThreeDotsLabs/watermill/message"
)

// Publisher is the output port the application depends on to emit events. The
// application imports this interface, not sarama - so a test can pass a fake.
type Publisher interface {
	PublishMessage(ctx context.Context, topic Topic, key string, payload any) error
}

// Message is a publish request carrying optional Kafka headers. The consumer's
// dead-letter path and the DLQ replay tool use it to preserve triage metadata;
// domain publishers keep using the smaller Publisher port.
type Message struct {
	Topic   Topic
	Key     string
	Payload any
	Headers map[string]string
}

// HeaderPublisher is the extended publish port used when metadata must travel
// with the payload. Producer implements it.
type HeaderPublisher interface {
	Publish(ctx context.Context, msg Message) error
}

var _ HeaderPublisher = (*Producer)(nil)

// Producer is the Kafka implementation of Publisher.
type Producer struct {
	pub message.Publisher
	m   metrics.Metrics
}

func NewProducer(brokers []string, m metrics.Metrics) (*Producer, error) {
	return NewProducerWithAuth(brokers, Auth{}, m)
}

func NewProducerWithAuth(brokers []string, auth Auth, m metrics.Metrics) (*Producer, error) {
	cfg := wkafka.DefaultSaramaSyncPublisherConfig()
	// V2_1_0_0 keeps the client on non-flexible protocol encodings, which every
	// Kafka-compatible broker (including Redpanda in dev mode) accepts.
	cfg.Version = sarama.V2_1_0_0
	applyAuth(cfg, auth)

	pub, err := wkafka.NewPublisher(
		wkafka.PublisherConfig{
			Brokers:               brokers,
			Marshaler:             wkafka.NewWithPartitioningMarshaler(partitionByKey),
			OverwriteSaramaConfig: cfg,
		},
		watermillLogAdapter(),
	)
	if err != nil {
		return nil, fmt.Errorf("new kafka publisher: %w", err)
	}
	return &Producer{pub: pub, m: m}, nil
}

// partitionByKey sends all messages sharing a key to the same partition, so events
// for one order stay ordered.
func partitionByKey(topic string, msg *message.Message) (string, error) {
	return msg.Metadata.Get("key"), nil
}

func (p *Producer) PublishMessage(ctx context.Context, topic Topic, key string, payload any) error {
	return p.Publish(ctx, Message{Topic: topic, Key: key, Payload: payload})
}

// Publish emits msg, copying its headers into Kafka record headers. It is the
// same synchronous path PublishMessage uses, so a successful return means the
// broker acknowledged the message.
func (p *Producer) Publish(ctx context.Context, m Message) error {
	start := time.Now()
	err := p.publish(ctx, m)
	status := metrics.StatusSuccess
	if err != nil {
		status = metrics.StatusError
	}
	p.m.IncKafkaPublished(m.Topic.String(), status)
	p.m.ObserveKafkaPublishLatency(m.Topic.String(), time.Since(start))
	return err
}

func (p *Producer) publish(ctx context.Context, m Message) error {
	body, err := marshalPayload(m.Payload)
	if err != nil {
		return err
	}
	msg := message.NewMessage(watermill.NewUUID(), body)
	msg.SetContext(ctx)
	// Headers are stored as metadata by the Watermill marshaler, which turns them
	// back into Kafka record headers. Set the partition key last so a header can
	// never override it.
	for k, v := range m.Headers {
		msg.Metadata.Set(k, v)
	}
	msg.Metadata.Set("key", m.Key)
	if err := p.pub.Publish(m.Topic.String(), msg); err != nil {
		return fmt.Errorf("publish to %s: %w", m.Topic, err)
	}
	return nil
}

func marshalPayload(payload any) ([]byte, error) {
	switch v := payload.(type) {
	case []byte:
		return v, nil
	default:
		body, err := json.Marshal(payload)
		if err != nil {
			return nil, fmt.Errorf("marshal event: %w", err)
		}
		return body, nil
	}
}

func (p *Producer) Close() error { return p.pub.Close() }
