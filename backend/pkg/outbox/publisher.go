package outbox

import (
	"context"
	"crypto/rand"
	"time"

	"github.com/jackc/pgx/v5"
)

// Publisher stores messages in the outbox. It is used by a domain's repository
// inside the same transaction that persists the aggregate, giving an atomic
// state-change-plus-event write.
type Publisher struct {
	store Store
	now   func() time.Time
}

// NewPublisher builds a Publisher over the given Store.
func NewPublisher(store Store) *Publisher {
	return &Publisher{store: store, now: time.Now}
}

// Send stores msg as a pending record within tx. The whole thing rolls back with
// tx if the surrounding transaction fails.
func (p *Publisher) Send(ctx context.Context, msg Message, tx pgx.Tx) error {
	return p.store.AddRecordTx(ctx, Record{
		ID:        rand.Text(),
		Message:   msg,
		State:     PendingDelivery,
		CreatedOn: p.now().UTC(),
	}, tx)
}
