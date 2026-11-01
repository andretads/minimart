// Package outbox wires the payment domain to the reusable transactional outbox:
// the repository writes each payment's domain events into the shared outbox
// table, and the Relay (run by the worker role) dispatches them to Kafka.
package outbox

import (
	"context"
	"crypto/rand"
	"minimart/pkg/kafka"
	"os"
	"time"

	outboxpkg "minimart/pkg/outbox"
)

// Settings configures the payment outbox relay. Values come from conf, so this
// package stays free of environment reads and is easy to test.
type Settings struct {
	ProcessInterval           time.Duration
	LockCheckerInterval       time.Duration
	CleanupWorkerInterval     time.Duration
	MaxLockTimeDuration       time.Duration
	MessagesRetentionDuration time.Duration
	// MaxSendAttempts caps delivery attempts per record; zero retries forever.
	MaxSendAttempts int
}

// Relay runs the outbox dispatcher for the payment domain. It is a long-running
// background task started by the worker role via worker.AttachBackground.
type Relay struct {
	dispatcher *outboxpkg.Dispatcher
}

// NewRelay builds the dispatcher over the shared outbox store. machineID
// identifies this process's record locks and must be stable for the process's
// lifetime (MachineID provides one).
func NewRelay(store outboxpkg.Store, publisher kafka.Publisher, machineID string, s Settings) *Relay {
	settings := outboxpkg.DispatcherSettings{
		ProcessInterval:           s.ProcessInterval,
		LockCheckerInterval:       s.LockCheckerInterval,
		CleanupWorkerInterval:     s.CleanupWorkerInterval,
		MaxLockTimeDuration:       s.MaxLockTimeDuration,
		MessagesRetentionDuration: s.MessagesRetentionDuration,
		RetryPolicy: outboxpkg.RetryPolicy{
			MaxSendAttemptsEnabled: s.MaxSendAttempts > 0,
			MaxSendAttempts:        s.MaxSendAttempts,
		},
	}
	return &Relay{
		dispatcher: outboxpkg.NewDispatcher(store, kafka.NewOutboxBroker(publisher), settings, machineID),
	}
}

// Run blocks until ctx is cancelled, relaying pending payment events to Kafka.
func (r *Relay) Run(ctx context.Context) { r.dispatcher.Run(ctx) }

// MachineID returns a stable identifier for this process's outbox locks: the
// hostname, falling back to a random value when it is unavailable.
func MachineID() string {
	if h, err := os.Hostname(); err == nil && h != "" {
		return h
	}
	return rand.Text()
}
