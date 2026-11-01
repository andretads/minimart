// Package tests spins up the real dependencies integration tests need.
//
// It exists so integration tests are self-contained: they start their own
// MongoDB (and Kafka and Postgres, in the sibling files) with
// testcontainers-go instead of depending on a pre-provisioned docker-compose
// service. Nothing here imports "testing"; TestMain drives it, which keeps the
// testing flag registration out of a non-test package.
package tests

import (
	"context"
	"fmt"

	"github.com/testcontainers/testcontainers-go/modules/mongodb"
)

// mongoImage matches the major version docker-compose serves locally, so tests
// and local development exercise the same MongoDB.
const mongoImage = "mongo:7"

// Mongo is a running MongoDB container. Unlike Postgres there are no
// migrations to apply: the notification adapter creates its own indexes when
// the module boots.
type Mongo struct {
	Container *mongodb.MongoDBContainer
	URI       string
}

// StartMongo starts an ephemeral MongoDB container, waits until it accepts
// connections, and returns the container plus a connection URI for it. Call
// Close to stop it.
func StartMongo(ctx context.Context) (*Mongo, error) {
	container, err := mongodb.Run(ctx, mongoImage)
	if err != nil {
		return nil, fmt.Errorf("start mongo container: %w", err)
	}

	uri, err := container.ConnectionString(ctx)
	if err != nil {
		_ = container.Terminate(ctx)
		return nil, fmt.Errorf("mongo connection string: %w", err)
	}

	return &Mongo{Container: container, URI: uri}, nil
}

// Close stops the container. The testcontainers Ryuk reaper also removes
// containers left behind by a crashed test run.
func (m *Mongo) Close(ctx context.Context) error {
	return m.Container.Terminate(ctx)
}
