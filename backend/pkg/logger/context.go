package logger

import (
	"context"
	"crypto/rand"
)

// requestIDKey is the context key under which a request's correlation ID is
// stored.
type requestIDKey struct{}

// WithRequestID returns a context carrying the request's correlation ID.
func WithRequestID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, requestIDKey{}, id)
}

// RequestIDFromContext returns the request's correlation ID, or "" when the
// context carries none.
func RequestIDFromContext(ctx context.Context) string {
	id, _ := ctx.Value(requestIDKey{}).(string)
	return id
}

// newRequestID returns a cryptographically random request identifier, matching
// the scheme the application layer uses for entity IDs.
func newRequestID() string {
	return rand.Text()
}
