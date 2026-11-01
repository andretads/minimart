// Package resilience provides a small in-process circuit breaker used to stop
// hammering a degraded dependency (Redis) and fail fast instead.
package resilience

import (
	"context"
	"errors"
	"sync"
	"time"
)

// ErrOpen is returned when the breaker is open and the call is not attempted.
var ErrOpen = errors.New("circuit breaker is open")

// State is the breaker's current state.
type State string

const (
	StateClosed   State = "closed"
	StateOpen     State = "open"
	StateHalfOpen State = "half_open"
)

// Config tunes the breaker.
type Config struct {
	FailureThreshold int
	OpenDuration     time.Duration
	HalfOpenMax      int
	SuccessThreshold int
}

// Breaker is a thread-safe circuit breaker.
type Breaker struct {
	mu               sync.Mutex
	name             string
	cfg              Config
	state            State
	failures         int
	halfOpenInFlight int
	successes        int
	openedAt         time.Time
}

// New creates a closed breaker.
func New(name string, cfg Config) *Breaker {
	if cfg.FailureThreshold <= 0 {
		cfg.FailureThreshold = 5
	}
	if cfg.OpenDuration <= 0 {
		cfg.OpenDuration = 10 * time.Second
	}
	if cfg.HalfOpenMax <= 0 {
		cfg.HalfOpenMax = 1
	}
	if cfg.SuccessThreshold <= 0 {
		cfg.SuccessThreshold = 2
	}
	return &Breaker{name: name, cfg: cfg, state: StateClosed}
}

// Name returns the breaker's label.
func (b *Breaker) Name() string { return b.name }

// State returns the current state, transitioning open -> half-open when the open
// duration has elapsed.
func (b *Breaker) State() State {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.transitionLocked(time.Now())
	return b.state
}

// Execute runs fn if the breaker allows it, updating state from the result.
func (b *Breaker) Execute(ctx context.Context, fn func(context.Context) error) error {
	if err := b.beforeCall(time.Now()); err != nil {
		return err
	}
	err := fn(ctx)
	b.afterCall(time.Now(), err)
	return err
}

func (b *Breaker) beforeCall(now time.Time) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.transitionLocked(now)
	switch b.state {
	case StateOpen:
		return ErrOpen
	case StateHalfOpen:
		if b.halfOpenInFlight >= b.cfg.HalfOpenMax {
			return ErrOpen
		}
		b.halfOpenInFlight++
	}
	return nil
}

func (b *Breaker) afterCall(now time.Time, err error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.state == StateHalfOpen {
		b.halfOpenInFlight--
		if b.halfOpenInFlight < 0 {
			b.halfOpenInFlight = 0
		}
	}
	if err != nil {
		b.failures++
		b.successes = 0
		if b.state == StateHalfOpen {
			b.state = StateOpen
			b.openedAt = now
			b.halfOpenInFlight = 0
		} else if b.failures >= b.cfg.FailureThreshold {
			b.state = StateOpen
			b.openedAt = now
		}
		return
	}
	if b.state == StateHalfOpen {
		b.successes++
		if b.successes >= b.cfg.SuccessThreshold {
			b.state = StateClosed
			b.failures = 0
			b.successes = 0
			b.halfOpenInFlight = 0
		}
	} else {
		b.failures = 0
	}
}

func (b *Breaker) transitionLocked(now time.Time) {
	if b.state == StateOpen && now.Sub(b.openedAt) >= b.cfg.OpenDuration {
		b.state = StateHalfOpen
		b.halfOpenInFlight = 0
		b.successes = 0
	}
}
