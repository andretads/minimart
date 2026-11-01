//go:build integration

// Package test holds the order domain's integration tests.
package test

import (
	"minimart/pkg/tests"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	idemredis "minimart/pkg/idempotency/redis"
	ratelimitredis "minimart/pkg/ratelimit/redis"
	redisclient "minimart/pkg/redis"
)

// TestRedisAdapters_UseSeparateNamespaces verifies that the rate limiter and
// idempotency store use different Redis key namespaces. This prevents a
// collision where the limiter's INCR integer would corrupt the idempotency
// store's JSON entry, causing "order: pedido em andamento" false positives.
func TestRedisAdapters_UseSeparateNamespaces(t *testing.T) {
	// Start a dedicated Redis container for this test.
	rdbContainer, err := tests.StartRedis(t.Context())
	require.NoError(t, err)
	t.Cleanup(func() {
		_ = rdbContainer.Close(t.Context())
	})

	rdb, err := redisclient.New(rdbContainer.URL)
	require.NoError(t, err)
	t.Cleanup(func() {
		_ = rdb.Close()
	})

	limiter := ratelimitredis.New(rdb.Client, 100, 100, time.Minute)
	store := idemredis.New(rdb.Client)

	ctx := t.Context()
	key := "shared-" + uniqueOrderID()

	_, err = limiter.Allow(ctx, key)
	require.NoError(t, err)

	// The idempotency store must not see the limiter's counter.
	rec, found, err := store.Get(ctx, key)
	require.NoError(t, err)
	assert.False(t, found)
	assert.Empty(t, rec)

	// Reserving the same logical key must succeed.
	ok, err := store.Reserve(ctx, key, "fp", time.Minute)
	require.NoError(t, err)
	assert.True(t, ok)

	rec, found, err = store.Get(ctx, key)
	require.NoError(t, err)
	assert.True(t, found)
	assert.Equal(t, "fp", rec.Fingerprint)

	// The limiter can keep counting without corrupting the idempotency entry.
	_, err = limiter.Allow(ctx, key)
	require.NoError(t, err)

	rec, found, err = store.Get(ctx, key)
	require.NoError(t, err)
	assert.True(t, found)
	assert.Equal(t, "fp", rec.Fingerprint)
}
