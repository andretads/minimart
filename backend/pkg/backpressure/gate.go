// Package backpressure provides a small pause/resume gate that Kafka consumers
// consult before reading another message. It is the worker-level mechanism for
// keeping consumption below the system's capacity: when a dependency (the write
// pool, Redis, a provider) saturates, the gate pauses so the queue absorbs the
// excess instead of amplifying the overload.
package backpressure

import (
	"context"
	"log/slog"
	"sync"
)

// Gate is a pause/resume switch. Wait blocks while paused and returns only when
// the gate is resumed or the context is cancelled.
type Gate struct {
	mu     sync.Mutex
	paused bool
	wake   chan struct{}
}

// NewGate builds an unpaused gate.
func NewGate() *Gate {
	return &Gate{wake: make(chan struct{})}
}

// Pause marks the gate paused. A reason is logged at info level. It is safe to
// call while already paused; the pause window is extended.
func (g *Gate) Pause(reason string) {
	g.mu.Lock()
	if !g.paused {
		g.paused = true
		g.wake = make(chan struct{})
		slog.Info("backpressure gate paused", "reason", reason)
	}
	g.mu.Unlock()
}

// Resume marks the gate running. It is a no-op when already running.
func (g *Gate) Resume() {
	g.mu.Lock()
	if g.paused {
		g.paused = false
		close(g.wake)
		slog.Info("backpressure gate resumed")
	}
	g.mu.Unlock()
}

// Paused reports whether the gate is currently paused.
func (g *Gate) Paused() bool {
	g.mu.Lock()
	defer g.mu.Unlock()
	return g.paused
}

// Wait blocks while the gate is paused. It returns ctx.Err() when the context
// is cancelled.
func (g *Gate) Wait(ctx context.Context) error {
	for {
		g.mu.Lock()
		if !g.paused {
			g.mu.Unlock()
			return nil
		}
		ch := g.wake
		g.mu.Unlock()

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ch:
		}
	}
}
