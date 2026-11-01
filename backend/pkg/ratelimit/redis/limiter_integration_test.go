//go:build integration

package redis

import (
	"minimart/pkg/redis"
	"minimart/pkg/tests"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTokenBucket_BurstThenThrottle(t *testing.T) {
	rdbContainer, err := tests.StartRedis(t.Context())
	require.NoError(t, err)
	t.Cleanup(func() { _ = rdbContainer.Close(t.Context()) })

	rdb, err := redis.New(rdbContainer.URL)
	require.NoError(t, err)
	t.Cleanup(func() { _ = rdb.Close() })

	lim := New(rdb.Client, 3, 3, time.Second)
	ctx := t.Context()
	key := "tb-burst-" + uniqueSuffix(t)

	for i := range 3 {
		res, lerr := lim.Allow(ctx, key)
		require.NoError(t, lerr)
		assert.True(t, res.Allowed, "request %d should be allowed", i)
	}
	res, lerr := lim.Allow(ctx, key)
	require.NoError(t, lerr)
	assert.False(t, res.Allowed)
	assert.Greater(t, res.RetryAfter, time.Duration(0))
}

func TestTokenBucket_Refills(t *testing.T) {
	rdbContainer, err := tests.StartRedis(t.Context())
	require.NoError(t, err)
	t.Cleanup(func() { _ = rdbContainer.Close(t.Context()) })

	rdb, err := redis.New(rdbContainer.URL)
	require.NoError(t, err)
	t.Cleanup(func() { _ = rdb.Close() })

	lim := New(rdb.Client, 2, 2, 200*time.Millisecond)
	ctx := t.Context()
	key := "tb-refill-" + uniqueSuffix(t)

	for range 2 {
		res, lerr := lim.Allow(ctx, key)
		require.NoError(t, lerr)
		assert.True(t, res.Allowed)
	}

	time.Sleep(250 * time.Millisecond)
	res, lerr := lim.Allow(ctx, key)
	require.NoError(t, lerr)
	assert.True(t, res.Allowed, "bucket should have refilled")
}

func TestTokenBucket_RemainingClampsAtZero(t *testing.T) {
	rdbContainer, err := tests.StartRedis(t.Context())
	require.NoError(t, err)
	t.Cleanup(func() { _ = rdbContainer.Close(t.Context()) })

	rdb, err := redis.New(rdbContainer.URL)
	require.NoError(t, err)
	t.Cleanup(func() { _ = rdb.Close() })

	lim := New(rdb.Client, 1, 1, time.Second)
	ctx := t.Context()
	key := "tb-remaining-" + uniqueSuffix(t)

	res, lerr := lim.Allow(ctx, key)
	require.NoError(t, lerr)
	assert.True(t, res.Allowed)
	assert.Equal(t, 0, res.Remaining)

	res, lerr = lim.Allow(ctx, key)
	require.NoError(t, lerr)
	assert.False(t, res.Allowed)
	assert.Equal(t, 0, res.Remaining)
}

func TestTokenBucket_IdleKeyExpires(t *testing.T) {
	rdbContainer, err := tests.StartRedis(t.Context())
	require.NoError(t, err)
	t.Cleanup(func() { _ = rdbContainer.Close(t.Context()) })

	rdb, err := redis.New(rdbContainer.URL)
	require.NoError(t, err)
	t.Cleanup(func() { _ = rdb.Close() })

	lim := New(rdb.Client, 1, 1, 50*time.Millisecond)
	ctx := t.Context()
	key := "tb-expire-" + uniqueSuffix(t)

	res, lerr := lim.Allow(ctx, key)
	require.NoError(t, lerr)
	assert.True(t, res.Allowed)

	time.Sleep(120 * time.Millisecond)
	res, lerr = lim.Allow(ctx, key)
	require.NoError(t, lerr)
	assert.True(t, res.Allowed, "fresh bucket after expiry")
}

func uniqueSuffix(t *testing.T) string {
	t.Helper()
	return t.Name() + "-" + time.Now().Format("150405.000000")
}
