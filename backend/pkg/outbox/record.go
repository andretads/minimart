// Package outbox implements the transactional outbox pattern. A domain writes a
// message to the outbox table in the same database transaction that changes its
// aggregate state, so the event and the state change either both commit or both
// roll back. A Dispatcher later relays undelivered messages to a message broker
// at least once.
//
// The design follows Panayiotis Kritiotis' reference implementation
// (github.com/pkritiotis/go-outbox), adapted to pgx and this repository's
// conventions: polling via a ticker, row locking so concurrent dispatchers do not
// double-publish, an optional maximum-attempts cap, a lock unlocker for crashed
// dispatchers, and a retention cleaner.
package outbox

import "time"

// Message is the broker-agnostic payload stored in the outbox and later published.
// Key is the broker partitioning key (MiniMart keys by aggregate ID so per-order
// events stay ordered); Body is the already-serialized payload.
type Message struct {
	Key   string
	Body  []byte
	Topic string
}

// Record is one row of the outbox table.
type Record struct {
	ID               string
	Message          Message
	State            RecordState
	CreatedOn        time.Time
	LockID           *string
	LockedOn         *time.Time
	ProcessedOn      *time.Time
	NumberOfAttempts int
	LastAttemptOn    *time.Time
	Error            *string
}

// RecordState is the delivery state of a Record.
type RecordState int

const (
	// PendingDelivery is the initial state of every record.
	PendingDelivery RecordState = iota
	// Delivered means the message reached the broker at least once.
	Delivered
	// MaxAttemptsReached means delivery kept failing and the record was abandoned.
	MaxAttemptsReached
)
