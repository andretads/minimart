package authz

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"sync"
	"time"

	goredis "github.com/redis/go-redis/v9"

	"golang.org/x/sync/singleflight"
)

// cacheWriteTimeout bounds a Redis cache write so it never blocks the caller
// for long, even when the request context has already been canceled.
const cacheWriteTimeout = 5 * time.Second

type RedisClient interface {
	Get(ctx context.Context, key string) *goredis.StringCmd
	Set(ctx context.Context, key string, value any, ttl time.Duration) *goredis.StatusCmd
}

type CachedStore struct {
	store    PermissionStore
	redis    RedisClient
	ttl      time.Duration
	localTTL time.Duration
	logger   *slog.Logger

	mu    sync.RWMutex
	local map[string]localPermissionEntry

	sf singleflight.Group
}

type localPermissionEntry struct {
	perms   []string
	expires time.Time
}

func NewCachedStore(store PermissionStore, redisClient RedisClient, ttl time.Duration, logger *slog.Logger) *CachedStore {
	return NewCachedStoreWithOptions(store, redisClient, ttl, time.Minute, logger)
}

func NewCachedStoreWithOptions(store PermissionStore, redisClient RedisClient, ttl, localTTL time.Duration, logger *slog.Logger) *CachedStore {
	if localTTL <= 0 {
		localTTL = time.Minute
	}
	return &CachedStore{
		store:    store,
		redis:    redisClient,
		ttl:      ttl,
		localTTL: localTTL,
		logger:   logger,
		local:    make(map[string]localPermissionEntry),
	}
}

func (c *CachedStore) PermissionsForAccount(ctx context.Context, accountID string) ([]string, error) {
	if c.redis == nil {
		return c.store.PermissionsForAccount(ctx, accountID)
	}

	if perms, ok := c.localPermissions(accountID); ok {
		return perms, nil
	}

	v, err, _ := c.sf.Do(accountID, func() (any, error) {
		if perms, ok := c.localPermissions(accountID); ok {
			return perms, nil
		}

		key := "rbac:perm:" + accountID
		data, err := c.redis.Get(ctx, key).Result()
		if err == nil && data != "" {
			var perms []string
			if json.Unmarshal([]byte(data), &perms) == nil {
				c.storeLocalPermissions(accountID, perms)
				return cloneStrings(perms), nil
			}
		}

		perms, err := c.store.PermissionsForAccount(ctx, accountID)
		if err != nil {
			return nil, err
		}

		cacheCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), cacheWriteTimeout)
		defer cancel()
		b, _ := json.Marshal(perms)
		if err := c.redis.Set(cacheCtx, key, string(b), c.ttl).Err(); err != nil {
			c.logger.WarnContext(cacheCtx, "failed to cache permissions", "error", err, "account_id", accountID)
		}

		c.storeLocalPermissions(accountID, perms)
		return cloneStrings(perms), nil
	})
	if err != nil {
		return nil, err
	}
	perms, ok := v.([]string)
	if !ok {
		return nil, errors.New("unexpected permission cache type")
	}
	return cloneStrings(perms), nil
}

// localPermissions returns a live local cache entry for the account. Expired
// entries are removed instead of lingering until the process restarts.
func (c *CachedStore) localPermissions(accountID string) ([]string, bool) {
	now := time.Now()
	c.mu.RLock()
	entry, ok := c.local[accountID]
	c.mu.RUnlock()
	if !ok {
		return nil, false
	}
	if now.After(entry.expires) {
		c.deleteLocalPermissions(accountID)
		return nil, false
	}
	return cloneStrings(entry.perms), true
}

// deleteLocalPermissions removes the entry if it is still expired; a
// concurrent refresh may have replaced it in the meantime.
func (c *CachedStore) deleteLocalPermissions(accountID string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	entry, ok := c.local[accountID]
	if ok && time.Now().After(entry.expires) {
		delete(c.local, accountID)
	}
}

func (c *CachedStore) storeLocalPermissions(accountID string, perms []string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.local[accountID] = localPermissionEntry{
		perms:   cloneStrings(perms),
		expires: time.Now().Add(c.localTTL),
	}
}

func cloneStrings(in []string) []string {
	out := make([]string, len(in))
	copy(out, in)
	return out
}

var _ PermissionStore = (*CachedStore)(nil)
