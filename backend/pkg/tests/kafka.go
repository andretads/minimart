// Package tests spins up the real dependencies integration tests need.
//
// It exists so integration tests are self-contained: they start their own
// Kafka (via testcontainers-go kafka module) and apply the same
// migrations/config the production binary expects. Nothing here imports "testing";
// TestMain drives it, which keeps the testing flag registration out of a non-test
// package.
package tests

import (
	"context"
	"fmt"

	kafkatc "github.com/testcontainers/testcontainers-go/modules/kafka"
)

// kafkaImage matches a recent Confluent Local image compatible with KRaft.
const kafkaImage = "confluentinc/confluent-local:7.5.0"

// Kafka is a running Kafka container.
type Kafka struct {
	Container *kafkatc.KafkaContainer
	Brokers   []string
}

// StartKafka starts an ephemeral Kafka container and returns it with broker list.
func StartKafka(ctx context.Context) (*Kafka, error) {
	container, err := kafkatc.Run(ctx, kafkaImage)
	if err != nil {
		return nil, fmt.Errorf("start kafka container: %w", err)
	}

	brokers, err := container.Brokers(ctx)
	if err != nil {
		_ = container.Terminate(ctx)
		return nil, fmt.Errorf("kafka brokers: %w", err)
	}

	return &Kafka{Container: container, Brokers: brokers}, nil
}

// Close stops the container.
func (k *Kafka) Close(ctx context.Context) error {
	return k.Container.Terminate(ctx)
}
