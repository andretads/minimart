package outbox

import "context"

// RelayLock serializes outbox publish cycles across worker replicas.
type RelayLock interface {
	TryAcquire(ctx context.Context) (bool, error)
	Release(ctx context.Context) error
}

type noopRelayLock struct{}

func (noopRelayLock) TryAcquire(context.Context) (bool, error) { return true, nil }
func (noopRelayLock) Release(context.Context) error            { return nil }
