package kafka

import (
	"context"
	"log/slog"
	"time"

	"github.com/IBM/sarama"
)

// LagReader reads consumer lag for (topic, group) pairs.
type LagReader interface {
	Lag(ctx context.Context, topic Topic, group string) (int64, error)
}

// SaramaLagReader implements LagReader using Kafka admin APIs.
type SaramaLagReader struct {
	brokers []string
}

// NewSaramaLagReader builds a reader.
func NewSaramaLagReader(brokers []string) *SaramaLagReader {
	return &SaramaLagReader{brokers: brokers}
}

func (r *SaramaLagReader) Lag(_ context.Context, topic Topic, group string) (int64, error) {
	client, err := sarama.NewClient(r.brokers, sarama.NewConfig())
	if err != nil {
		return 0, err
	}
	defer func() { _ = client.Close() }()

	admin, err := sarama.NewClusterAdminFromClient(client)
	if err != nil {
		return 0, err
	}
	defer func() { _ = admin.Close() }()

	// Latest offsets per partition.
	partitions, err := client.Partitions(topic.String())
	if err != nil {
		return 0, err
	}

	committed, err := admin.ListConsumerGroupOffsets(group, map[string][]int32{topic.String(): partitions})
	if err != nil {
		return 0, err
	}

	var total int64
	for _, p := range partitions {
		latest, err := client.GetOffset(topic.String(), p, sarama.OffsetNewest)
		if err != nil {
			return 0, err
		}
		var comm int64
		if committed != nil && committed.Blocks != nil {
			if m, ok := committed.Blocks[topic.String()]; ok {
				if b, ok := m[p]; ok && b != nil {
					comm = b.Offset
				}
			}
		}
		if comm < 0 {
			comm = 0
		}
		if latest > comm {
			total += latest - comm
		}
	}
	return total, nil
}

// LagMonitor periodically samples lag for the given subscriptions and records
// it through the supplied callback. It runs until ctx is cancelled.
type LagMonitor struct {
	reader        LagReader
	subscriptions func() []Subscription
	record        func(topic string, group string, lag int64)
	interval      time.Duration
}

// NewLagMonitor builds a monitor.
func NewLagMonitor(reader LagReader, subscriptions func() []Subscription, record func(string, string, int64), interval time.Duration) *LagMonitor {
	if interval <= 0 {
		interval = 30 * time.Second
	}
	return &LagMonitor{
		reader:        reader,
		subscriptions: subscriptions,
		record:        record,
		interval:      interval,
	}
}

// Run blocks until ctx is cancelled.
func (m *LagMonitor) Run(ctx context.Context) {
	ticker := time.NewTicker(m.interval)
	defer ticker.Stop()
	for {
		m.sample(ctx)
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

func (m *LagMonitor) sample(ctx context.Context) {
	for _, s := range m.subscriptions() {
		lag, err := m.reader.Lag(ctx, s.Topic, s.Group)
		if err != nil {
			slog.Warn("consumer lag read failed", "topic", s.Topic.String(), "group", s.Group, "error", err)
			continue
		}
		if m.record != nil {
			m.record(s.Topic.String(), s.Group, lag)
		}
	}
}
