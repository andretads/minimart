// Package tests spins up the real dependencies integration tests need.
//
// It exists so integration tests are self-contained: they start their own Redis
// (like Postgres, Kafka, and MongoDB in the sibling files) with testcontainers-go
// instead of depending on a pre-provisioned docker-compose service. Nothing here
// imports "testing"; TestMain drives it, which keeps the testing flag
// registration out of a non-test package.
package tests

import (
	"context"
	"fmt"

	"github.com/testcontainers/testcontainers-go/modules/redis"
)

// redisImage matches the major version docker-compose serves locally, so tests
// and local development exercise the same Redis.
const redisImage = "redis:7-alpine"

// Redis is a running Redis container.
type Redis struct {
	Container *redis.RedisContainer
	URL       string
}

// StartRedis starts an ephemeral Redis container, waits until it accepts
// connections, and returns the container plus a redis:// connection URL. Call
// Close to stop it.
func StartRedis(ctx context.Context) (*Redis, error) {
	container, err := redis.Run(ctx, redisImage)
	if err != nil {
		return nil, fmt.Errorf("start redis container: %w", err)
	}

	url, err := container.ConnectionString(ctx)
	if err != nil {
		_ = container.Terminate(ctx)
		return nil, fmt.Errorf("redis connection string: %w", err)
	}

	return &Redis{Container: container, URL: url}, nil
}

// Close stops the container. The testcontainers Ryuk reaper also removes
// containers left behind by a crashed test run.
func (r *Redis) Close(ctx context.Context) error {
	return r.Container.Terminate(ctx)
}
