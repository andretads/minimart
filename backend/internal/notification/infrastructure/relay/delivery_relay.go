// Package relay runs the notification delivery dispatch loop in the worker role.
package relay

import (
	"context"
	"minimart/internal/notification/application"
	"time"
)

// Relay periodically claims due deliveries and dispatches them to their
// channel-specific Kafka topics.
type Relay struct {
	svc *application.Service
}

// New builds a relay over the notification application service.
func New(svc *application.Service) *Relay {
	return &Relay{svc: svc}
}

// Run blocks until ctx is cancelled, dispatching due deliveries.
func (r *Relay) Run(ctx context.Context) {
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()
	for {
		_, _ = r.svc.DispatchDueDeliveries(ctx, "notification-relay", 50)
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}
