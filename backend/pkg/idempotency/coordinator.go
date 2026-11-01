// Package idempotency also provides a small coordinator that encapsulates the
// repeated reserve/replay/complete/release flow shared by idempotent use cases.
// It is generic over the resource type produced by an operation.
package idempotency

import (
	"context"
	"errors"
	"log/slog"
	"time"
)

// Coordinator orchestrates an idempotency Store for a single resource type. It
// owns no business logic: the use case supplies how to load a resource by id or
// by key and which sentinel error to return while an operation is in flight.
type Coordinator[T any] struct {
	Store          Store
	ReservationTTL time.Duration
	CompletionTTL  time.Duration
	LoadByID       func(ctx context.Context, id string) (T, error)
	LoadByKey      func(ctx context.Context, key, fingerprint string) (T, bool, error)
	NotFoundErr    error
	InProgressErr  error
	ConflictErr    error
	OnMiss         func(ctx context.Context)
	OnConflict     func(ctx context.Context)
	OnInProgress   func(ctx context.Context)
	OnReplay       func(ctx context.Context)
	OnStoreError   func(ctx context.Context, err error)
}

func (c *Coordinator[T]) storeError(ctx context.Context, err error) {
	if c.OnStoreError != nil {
		c.OnStoreError(ctx, err)
		return
	}
	slog.WarnContext(ctx, "idempotency store unavailable", "error", err)
}

// Existing resolves a key to an already completed result. When the store is
// unavailable it falls back to LoadByKey.
func (c *Coordinator[T]) Existing(ctx context.Context, key, fingerprint string) (T, bool, error) {
	var zero T
	if c.Store == nil {
		return c.LoadByKey(ctx, key, fingerprint)
	}
	rec, found, err := c.Store.Get(ctx, key)
	if err != nil {
		c.storeError(ctx, err)
		return c.LoadByKey(ctx, key, fingerprint)
	}
	if !found {
		if c.OnMiss != nil {
			c.OnMiss(ctx)
		}
		return zero, false, nil
	}
	if rec.Fingerprint != fingerprint {
		if c.OnConflict != nil {
			c.OnConflict(ctx)
		}
		if c.ConflictErr != nil {
			return zero, false, c.ConflictErr
		}
		return zero, false, ErrConflict
	}
	if rec.ResourceID == "" {
		if c.OnInProgress != nil {
			c.OnInProgress(ctx)
		}
		if c.InProgressErr != nil {
			return zero, false, c.InProgressErr
		}
		return zero, false, nil
	}
	if c.OnReplay != nil {
		c.OnReplay(ctx)
	}
	item, err := c.LoadByID(ctx, rec.ResourceID)
	if err != nil {
		if c.NotFoundErr != nil && errors.Is(err, c.NotFoundErr) {
			return c.LoadByKey(ctx, key, fingerprint)
		}
		return zero, false, err
	}
	return item, true, nil
}

// ErrConflict is returned when a key is reused with a different fingerprint.
var ErrConflict = errors.New("idempotency: key reused with different parameters")

// Reserve claims a key. It fails open on store errors.
func (c *Coordinator[T]) Reserve(ctx context.Context, key, fingerprint string) bool {
	if c.Store == nil {
		return true
	}
	ok, err := c.Store.Reserve(ctx, key, fingerprint, c.ReservationTTL)
	if err != nil {
		c.storeError(ctx, err)
		return true
	}
	return ok
}

// Complete records a finished result. Failures are logged but not fatal.
func (c *Coordinator[T]) Complete(ctx context.Context, key, fingerprint, resourceID string) {
	if c.Store == nil {
		return
	}
	if err := c.Store.Complete(ctx, key, fingerprint, resourceID, c.CompletionTTL); err != nil {
		c.storeError(ctx, err)
	}
}

// Release drops a reservation. Failures are logged but not fatal.
func (c *Coordinator[T]) Release(ctx context.Context, key, fingerprint string) {
	if c.Store == nil {
		return
	}
	if err := c.Store.Release(ctx, key, fingerprint); err != nil {
		c.storeError(ctx, err)
	}
}

// LookupByKey delegates to the use case's authoritative fallback.
func (c *Coordinator[T]) LookupByKey(ctx context.Context, key, fingerprint string) (T, bool, error) {
	return c.LoadByKey(ctx, key, fingerprint)
}
