package metrics

import (
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

// kafkaMetrics carries the Kafka pipeline collectors: publishing, consuming,
// retries, dead letters, lag, and backpressure.
type kafkaMetrics struct {
	kafkaPublished       *prometheus.CounterVec
	kafkaPublishDur      *prometheus.HistogramVec
	kafkaConsumed        *prometheus.CounterVec
	kafkaConsumeDur      *prometheus.HistogramVec
	kafkaRetried         *prometheus.CounterVec
	kafkaDeadLettered    *prometheus.CounterVec
	kafkaConsumerRestart *prometheus.CounterVec
	kafkaConsumerLag     *prometheus.GaugeVec
	kafkaBackpressure    *prometheus.CounterVec
	kafkaConsumerPaused  *prometheus.GaugeVec
	kafkaDLQReplayParked *prometheus.CounterVec
}

// newKafkaMetrics builds and registers the Kafka collectors.
func newKafkaMetrics(reg prometheus.Registerer) kafkaMetrics {
	m := kafkaMetrics{
		// Dashboard: sum by (topic) (rate(minimart_kafka_messages_published_total[5m]))
		// Alert:     rate(minimart_kafka_messages_published_total{status="error"}[5m]) > 0.5
		kafkaPublished: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: namespace,
			Subsystem: "kafka",
			Name:      "messages_published_total",
			Help:      "Total number of Kafka messages published by topic and result.",
		}, []string{"topic", "status"}),

		// Dashboard: histogram_quantile(0.99, sum(rate(minimart_kafka_publish_duration_seconds_bucket[5m])) by (le, topic))
		kafkaPublishDur: prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Namespace: namespace,
			Subsystem: "kafka",
			Name:      "publish_duration_seconds",
			Help:      "Kafka publish duration in seconds.",
			Buckets:   prometheus.DefBuckets,
		}, []string{"topic"}),

		// Dashboard: sum by (topic, group) (rate(minimart_kafka_messages_consumed_total[5m]))
		// Alert:     rate(minimart_kafka_messages_consumed_total{status="error"}[5m]) > 0.5
		kafkaConsumed: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: namespace,
			Subsystem: "kafka",
			Name:      "messages_consumed_total",
			Help:      "Total number of Kafka messages consumed by topic, group, and result.",
		}, []string{"topic", "group", "status"}),

		// Dashboard: histogram_quantile(0.99, sum(rate(minimart_kafka_consume_duration_seconds_bucket[5m])) by (le, topic))
		kafkaConsumeDur: prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Namespace: namespace,
			Subsystem: "kafka",
			Name:      "consume_duration_seconds",
			Help:      "Kafka message handler duration in seconds.",
			Buckets:   prometheus.DefBuckets,
		}, []string{"topic", "group"}),

		// Dashboard: sum by (topic, group) (rate(minimart_kafka_retries_total[5m]))
		kafkaRetried: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: namespace,
			Subsystem: "kafka",
			Name:      "retries_total",
			Help:      "Total number of Kafka handler retries by topic and consumer group.",
		}, []string{"topic", "group"}),

		// Dashboard: sum by (topic, group, disposition) (rate(minimart_kafka_dead_lettered_total[5m]))
		// Alert:     rate(minimart_kafka_dead_lettered_total[5m]) > 0
		kafkaDeadLettered: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: namespace,
			Subsystem: "kafka",
			Name:      "dead_lettered_total",
			Help:      "Total number of Kafka messages routed to a dead-letter topic by topic, group, and disposition (permanent, exhausted).",
		}, []string{"topic", "group", "disposition"}),

		// Dashboard: sum by (topic, group) (rate(minimart_kafka_consumer_restarts_total[5m]))
		// Alert:     increase(minimart_kafka_consumer_restarts_total[15m]) > 0
		kafkaConsumerRestart: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: namespace,
			Subsystem: "kafka",
			Name:      "consumer_restarts_total",
			Help:      "Total number of Kafka consumer restarts by topic and consumer group.",
		}, []string{"topic", "group"}),

		kafkaConsumerLag: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: namespace,
			Subsystem: "kafka",
			Name:      "consumer_lag",
			Help:      "Consumer lag (latest minus committed offset) by topic and consumer group.",
		}, []string{"topic", "group"}),

		kafkaBackpressure: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: namespace,
			Subsystem: "kafka",
			Name:      "backpressure_waits_total",
			Help:      "Total number of times a consumer waited on the backpressure gate.",
		}, []string{"topic", "group"}),

		kafkaConsumerPaused: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: namespace,
			Subsystem: "kafka",
			Name:      "consumer_paused",
			Help:      "Whether a consumer group is currently paused (1) or running (0).",
		}, []string{"topic", "group"}),

		kafkaDLQReplayParked: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: namespace,
			Subsystem: "kafka",
			Name:      "dlq_replay_parked_total",
			Help:      "Total number of DLQ messages parked because the replay cap was reached.",
		}, []string{"topic"}),
	}
	reg.MustRegister(
		m.kafkaPublished,
		m.kafkaPublishDur,
		m.kafkaConsumed,
		m.kafkaConsumeDur,
		m.kafkaRetried,
		m.kafkaDeadLettered,
		m.kafkaConsumerRestart,
		m.kafkaConsumerLag,
		m.kafkaBackpressure,
		m.kafkaConsumerPaused,
		m.kafkaDLQReplayParked,
	)
	return m
}

func (m *kafkaMetrics) IncKafkaPublished(topic, status string) {
	m.kafkaPublished.WithLabelValues(topic, status).Inc()
}

func (m *kafkaMetrics) ObserveKafkaPublishLatency(topic string, d time.Duration) {
	m.kafkaPublishDur.WithLabelValues(topic).Observe(d.Seconds())
}

func (m *kafkaMetrics) IncKafkaConsumed(topic, group, status string) {
	m.kafkaConsumed.WithLabelValues(topic, group, status).Inc()
}

func (m *kafkaMetrics) ObserveKafkaConsumeLatency(topic, group string, d time.Duration) {
	m.kafkaConsumeDur.WithLabelValues(topic, group).Observe(d.Seconds())
}

func (m *kafkaMetrics) IncKafkaRetried(topic, group string) {
	m.kafkaRetried.WithLabelValues(topic, group).Inc()
}

func (m *kafkaMetrics) IncKafkaDeadLettered(topic, group, disposition string) {
	m.kafkaDeadLettered.WithLabelValues(topic, group, disposition).Inc()
}

func (m *kafkaMetrics) IncKafkaConsumerRestart(topic, group string) {
	m.kafkaConsumerRestart.WithLabelValues(topic, group).Inc()
}

func (m *kafkaMetrics) SetKafkaConsumerLag(topic, group string, lag int64) {
	m.kafkaConsumerLag.WithLabelValues(topic, group).Set(float64(lag))
}

func (m *kafkaMetrics) IncKafkaBackpressureWait(topic, group string) {
	m.kafkaBackpressure.WithLabelValues(topic, group).Inc()
}

func (m *kafkaMetrics) SetKafkaConsumerPaused(topic, group string, paused bool) {
	v := 0.0
	if paused {
		v = 1.0
	}
	m.kafkaConsumerPaused.WithLabelValues(topic, group).Set(v)
}

func (m *kafkaMetrics) IncKafkaDLQReplayParked(topic string) {
	m.kafkaDLQReplayParked.WithLabelValues(topic).Inc()
}
