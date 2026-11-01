package authz

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"sync"
	"testing"
	"time"

	goredis "github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/goleak"
)

func TestMain(m *testing.M) {
	goleak.VerifyTestMain(m)
}

type fakePermStore struct {
	mu    sync.Mutex
	perms map[string][]string
	err   error
}

func newFakePermStore() *fakePermStore {
	return &fakePermStore{perms: map[string][]string{
		"acc-1": {"orders:read", "catalog:read"},
		"acc-2": {"payments:write"},
	}}
}

func (f *fakePermStore) PermissionsForAccount(_ context.Context, accountID string) ([]string, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.err != nil {
		return nil, f.err
	}
	if perms, ok := f.perms[accountID]; ok {
		return perms, nil
	}
	return []string{}, nil
}

func (f *fakePermStore) SetErr(err error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.err = err
}

type fakeRedisClient struct {
	mu   sync.Mutex
	data map[string]string
	err  error
	sets []redisSetCall
	gets []string
}

type redisSetCall struct {
	key   string
	value string
	ttl   time.Duration
}

func newFakeRedisClient() *fakeRedisClient {
	return &fakeRedisClient{data: map[string]string{}}
}

func (f *fakeRedisClient) Get(_ context.Context, key string) *goredis.StringCmd {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.gets = append(f.gets, key)
	val, ok := f.data[key]
	cmd := goredis.NewStringCmd(context.Background())
	if !ok {
		cmd.SetErr(errors.New("redis: nil"))
		return cmd
	}
	cmd.SetVal(val)
	return cmd
}

func (f *fakeRedisClient) Set(_ context.Context, key string, value any, ttl time.Duration) *goredis.StatusCmd {
	f.mu.Lock()
	defer f.mu.Unlock()
	str, _ := value.(string)
	f.sets = append(f.sets, redisSetCall{key: key, value: str, ttl: ttl})
	f.data[key] = str
	cmd := goredis.NewStatusCmd(context.Background())
	if f.err != nil {
		cmd.SetErr(f.err)
		return cmd
	}
	cmd.SetVal("OK")
	return cmd
}

func (f *fakeRedisClient) SetErr(err error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.err = err
}

func (f *fakeRedisClient) Gets() []string {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := make([]string, len(f.gets))
	copy(out, f.gets)
	return out
}

func (f *fakeRedisClient) Sets() []redisSetCall {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := make([]redisSetCall, len(f.sets))
	copy(out, f.sets)
	return out
}

func newTestLogger() *slog.Logger {
	return slog.New(slog.DiscardHandler)
}

func TestCachedStore_ExpiredLocalEntryIsEvicted(t *testing.T) {
	t.Parallel()

	store := newFakePermStore()
	redis := newFakeRedisClient()
	logger := newTestLogger()
	cached := NewCachedStoreWithOptions(store, redis, 10*time.Minute, time.Nanosecond, logger)

	ctx := t.Context()
	_, err := cached.PermissionsForAccount(ctx, "acc-1")
	require.NoError(t, err)

	cached.mu.RLock()
	_, present := cached.local["acc-1"]
	cached.mu.RUnlock()
	require.True(t, present, "entry should be stored after first fetch")

	require.Eventually(t, func() bool {
		if _, ok := cached.localPermissions("acc-1"); ok {
			return false
		}
		cached.mu.RLock()
		defer cached.mu.RUnlock()
		_, present = cached.local["acc-1"]
		return !present
	}, 500*time.Millisecond, 10*time.Millisecond, "expired local entry should be removed from the cache map")
}

func TestCachedStore_CacheMissFetchesFromStoreAndWritesCache(t *testing.T) {
	t.Parallel()

	store := newFakePermStore()
	redis := newFakeRedisClient()
	logger := newTestLogger()
	cached := NewCachedStore(store, redis, 10*time.Minute, logger)

	ctx := t.Context()
	perms, err := cached.PermissionsForAccount(ctx, "acc-1")
	require.NoError(t, err)
	assert.Equal(t, []string{"orders:read", "catalog:read"}, perms)

	require.Eventually(t, func() bool {
		sets := redis.Sets()
		return len(sets) == 1 && sets[0].key == "rbac:perm:acc-1"
	}, 500*time.Millisecond, 10*time.Millisecond)

	sets := redis.Sets()
	require.Len(t, sets, 1)
	assert.Equal(t, "rbac:perm:acc-1", sets[0].key)

	var cachedPerms []string
	require.NoError(t, json.Unmarshal([]byte(sets[0].value), &cachedPerms))
	assert.Equal(t, []string{"orders:read", "catalog:read"}, cachedPerms)
	assert.Equal(t, 10*time.Minute, sets[0].ttl)
}

func TestCachedStore_CacheHitReturnsCachedValue(t *testing.T) {
	t.Parallel()

	store := newFakePermStore()
	redis := newFakeRedisClient()
	logger := newTestLogger()

	cachedPerms := []string{"cached:perm"}
	b, _ := json.Marshal(cachedPerms)
	redis.data["rbac:perm:acc-1"] = string(b)

	cached := NewCachedStore(store, redis, 10*time.Minute, logger)

	ctx := t.Context()
	perms, err := cached.PermissionsForAccount(ctx, "acc-1")
	require.NoError(t, err)
	assert.Equal(t, cachedPerms, perms)

	gets := redis.Gets()
	assert.Contains(t, gets, "rbac:perm:acc-1")

	require.Eventually(t, func() bool {
		return len(redis.Sets()) == 0
	}, 500*time.Millisecond, 10*time.Millisecond)
}

func TestCachedStore_CacheCorruptedValueFallsBackToStore(t *testing.T) {
	t.Parallel()

	store := newFakePermStore()
	redis := newFakeRedisClient()
	logger := newTestLogger()

	redis.data["rbac:perm:acc-1"] = "not-json"

	cached := NewCachedStore(store, redis, 10*time.Minute, logger)

	ctx := t.Context()
	perms, err := cached.PermissionsForAccount(ctx, "acc-1")
	require.NoError(t, err)
	assert.Equal(t, []string{"orders:read", "catalog:read"}, perms)

	require.Eventually(t, func() bool {
		return len(redis.Sets()) == 1
	}, 500*time.Millisecond, 10*time.Millisecond)
}

func TestCachedStore_StoreErrorPropagates(t *testing.T) {
	t.Parallel()

	store := newFakePermStore()
	errDB := errors.New("db down")
	store.SetErr(errDB)
	redis := newFakeRedisClient()
	logger := newTestLogger()

	cached := NewCachedStore(store, redis, 10*time.Minute, logger)

	ctx := t.Context()
	_, err := cached.PermissionsForAccount(ctx, "acc-1")
	require.Error(t, err)
	assert.ErrorIs(t, err, errDB)
}

func TestCachedStore_UnknownAccountReturnsEmptySlice(t *testing.T) {
	t.Parallel()

	store := newFakePermStore()
	redis := newFakeRedisClient()
	logger := newTestLogger()

	cached := NewCachedStore(store, redis, 10*time.Minute, logger)

	ctx := t.Context()
	perms, err := cached.PermissionsForAccount(ctx, "unknown-account")
	require.NoError(t, err)
	assert.Empty(t, perms)

	require.Eventually(t, func() bool {
		sets := redis.Sets()
		return len(sets) == 1 && sets[0].key == "rbac:perm:unknown-account"
	}, 500*time.Millisecond, 10*time.Millisecond)
}

func TestCachedStore_NilRedisClientUsesStoreDirectly(t *testing.T) {
	t.Parallel()

	store := newFakePermStore()
	logger := newTestLogger()

	cached := NewCachedStore(store, nil, 10*time.Minute, logger)

	ctx := t.Context()
	perms, err := cached.PermissionsForAccount(ctx, "acc-1")
	require.NoError(t, err)
	assert.Equal(t, []string{"orders:read", "catalog:read"}, perms)
}

func TestCachedStore_RedisGetErrorFallsBackToStore(t *testing.T) {
	t.Parallel()

	store := newFakePermStore()
	redis := newFakeRedisClient()
	redis.SetErr(errors.New("redis connection failed"))
	logger := newTestLogger()

	cached := NewCachedStore(store, redis, 10*time.Minute, logger)

	ctx := t.Context()
	perms, err := cached.PermissionsForAccount(ctx, "acc-1")
	require.NoError(t, err)
	assert.Equal(t, []string{"orders:read", "catalog:read"}, perms)

	require.Eventually(t, func() bool {
		return len(redis.Sets()) == 1
	}, 500*time.Millisecond, 10*time.Millisecond)
}

func TestCachedStore_RedisSetErrorLoggedButDoesNotBlock(t *testing.T) {
	t.Parallel()

	store := newFakePermStore()
	redis := newFakeRedisClient()
	redis.SetErr(errors.New("redis set failed"))
	logger := newTestLogger()

	cached := NewCachedStore(store, redis, 10*time.Minute, logger)

	ctx := t.Context()
	perms, err := cached.PermissionsForAccount(ctx, "acc-1")
	require.NoError(t, err)
	assert.Equal(t, []string{"orders:read", "catalog:read"}, perms)

	require.Eventually(t, func() bool {
		return len(redis.Sets()) == 1
	}, 500*time.Millisecond, 10*time.Millisecond)
}

func TestCachedStore_ConcurrentCallsNoGoroutineLeak(t *testing.T) {
	goleak.VerifyNone(t)

	store := newFakePermStore()
	redis := newFakeRedisClient()
	logger := newTestLogger()

	cached := NewCachedStore(store, redis, 10*time.Minute, logger)

	const workers = 20
	errs := make(chan error, workers)
	var wg sync.WaitGroup

	for range workers {
		wg.Go(func() {
			_, err := cached.PermissionsForAccount(t.Context(), "acc-1")
			errs <- err
		})
	}
	wg.Wait()
	close(errs)

	for err := range errs {
		require.NoError(t, err)
	}

	require.Eventually(t, func() bool {
		return len(redis.Sets()) > 0
	}, 500*time.Millisecond, 10*time.Millisecond)
}
