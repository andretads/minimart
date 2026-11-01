// Package idempotency provides a small store for idempotency keys and a stable
// fingerprint helper. A caller reserves a key before doing work, completes it
// with the resulting resource id, and replays the stored result when the same
// key arrives again. Reusing a key with different parameters is detected by
// comparing fingerprints.
//
// It is deliberately backend-agnostic: the Store port is backed by Redis in
// production (pkg/idempotency/redis), but the database remains the authoritative
// guard against duplicates, so a Redis outage degrades to database-only
// idempotency rather than corrupting behavior.
package idempotency

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"time"
)

// Record is a stored idempotency entry for one key.
type Record struct {
	// Fingerprint identifies the original request's parameters. A different
	// fingerprint for the same key means the key was reused with different
	// input, which callers must reject.
	Fingerprint string
	// ResourceID is the identifier of the completed result. It is empty while
	// the operation is still in flight (reserved but not completed).
	ResourceID string
}

// Store remembers idempotency keys. Every method may return an error when the
// backing store is unavailable; callers that treat the store as a cache decide
// whether to fail open (proceed, relying on the database) or closed.
type Store interface {
	// Get returns the stored record for key. found is false when the key has
	// never been seen.
	Get(ctx context.Context, key string) (rec Record, found bool, err error)
	// Reserve atomically claims key for fingerprint if no entry exists, giving
	// the caller exclusive right to process the operation. It returns false when
	// an entry already exists (an in-flight reservation or a completed result).
	Reserve(ctx context.Context, key, fingerprint string, ttl time.Duration) (bool, error)
	// Complete records the finished operation's resourceID under key, replacing
	// the reservation, and refreshes the entry's TTL.
	Complete(ctx context.Context, key, fingerprint, resourceID string, ttl time.Duration) error
	// Release removes a reservation so a later retry can proceed.
	Release(ctx context.Context, key, fingerprint string) error
}

// Fingerprint derives a stable, collision-resistant identifier for a request's
// idempotency-relevant parameters. Parts are length-prefixed before hashing so
// ["ab","c"] and ["a","bc"] produce different fingerprints. Callers must pass the
// parts in a fixed order.
func Fingerprint(parts ...string) string {
	h := sha256.New()
	for _, p := range parts {
		// Writes to a hash.Hash never fail, so the length prefix's error is safe
		// to drop.
		_, _ = fmt.Fprintf(h, "%d:", len(p))
		h.Write([]byte(p))
	}
	return hex.EncodeToString(h.Sum(nil))
}

// Header is the HTTP header carrying an idempotency key.
const Header = "Idempotency-Key"
