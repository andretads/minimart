package outbox

import "context"

// MessageBroker is the publish port the Dispatcher depends on. It is intentionally
// tiny so any broker can back it; a test can pass a fake.
type MessageBroker interface {
	Send(ctx context.Context, message Message) error
}
