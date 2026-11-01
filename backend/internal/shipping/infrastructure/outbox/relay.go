package outbox

import (
	"context"
	"crypto/rand"
	"minimart/pkg/kafka"
	"os"

	outboxpkg "minimart/pkg/outbox"
)

type Relay struct {
	dispatcher *outboxpkg.Dispatcher
}

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

func (r *Relay) Run(ctx context.Context) { r.dispatcher.Run(ctx) }

func MachineID() string {
	if h, err := os.Hostname(); err == nil && h != "" {
		return h
	}
	return rand.Text()
}
