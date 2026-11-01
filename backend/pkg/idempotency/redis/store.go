// Package redis is the Redis adapter of idempotency.Store. Each key is a single
// Redis string holding a JSON entry, so Reserve is an atomic SET NX and Get is a
// single GET.
package redis

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"minimart/pkg/idempotency"
	"time"

	goredis "github.com/redis/go-redis/v9"
)

// Store implements idempotency.Store over a go-redis client.
type Store struct {
	rdb *goredis.Client
}

var _ idempotency.Store = (*Store)(nil)

// New builds a Store over the given client.
func New(rdb *goredis.Client) *Store {
	return &Store{rdb: rdb}
}

// idempotencyPrefix is prepended to every idempotency key so the shared Redis
// instance cannot cross-contaminate with other adapters (rate limiting, authz).
const idempotencyPrefix = "idempotency:"

func (s *Store) key(k string) string {
	return idempotencyPrefix + k
}

// entry is the JSON value stored per key. ResourceID is omitted while a request
// holds an in-flight reservation.
type entry struct {
	Fingerprint string `json:"fingerprint"`
	ResourceID  string `json:"resource_id,omitempty"`
}

func (s *Store) Get(ctx context.Context, key string) (idempotency.Record, bool, error) {
	raw, err := s.rdb.Get(ctx, s.key(key)).Bytes()
	if errors.Is(err, goredis.Nil) {
		return idempotency.Record{}, false, nil
	}
	if err != nil {
		return idempotency.Record{}, false, err
	}
	var e entry
	if err := json.Unmarshal(raw, &e); err != nil {
		return idempotency.Record{}, false, fmt.Errorf("decode idempotency entry: %w", err)
	}
	return idempotency.Record{Fingerprint: e.Fingerprint, ResourceID: e.ResourceID}, true, nil
}

func (s *Store) Reserve(ctx context.Context, key, fingerprint string, ttl time.Duration) (bool, error) {
	raw, err := json.Marshal(entry{Fingerprint: fingerprint})
	if err != nil {
		return false, fmt.Errorf("encode idempotency entry: %w", err)
	}
	return s.rdb.SetNX(ctx, s.key(key), raw, ttl).Result()
}

func (s *Store) Complete(ctx context.Context, key, fingerprint, resourceID string, ttl time.Duration) error {
	raw, err := json.Marshal(entry{Fingerprint: fingerprint, ResourceID: resourceID})
	if err != nil {
		return fmt.Errorf("encode idempotency entry: %w", err)
	}
	return s.rdb.Set(ctx, s.key(key), raw, ttl).Err()
}

func (s *Store) Release(ctx context.Context, key, fingerprint string) error {
	// Compare-and-delete: only remove the reservation if it still belongs to
	// this request's fingerprint, so a loser cannot delete a winner's entry.
	script := `
if redis.call("GET", KEYS[1]) == false then
	return 0
end
local raw = redis.call("GET", KEYS[1])
if not raw then
	return 0
end
local ok, entry = pcall(cjson.decode, raw)
if not ok or entry.fingerprint ~= ARGV[1] then
	return 0
end
return redis.call("DEL", KEYS[1])
`
	_, err := s.rdb.Eval(ctx, script, []string{s.key(key)}, fingerprint).Result()
	return err
}
