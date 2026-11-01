package postgres

import (
	"context"
	"errors"
	"fmt"
	"minimart/internal/payment/domain"
	"minimart/internal/payment/infrastructure/postgres/reader"
	"minimart/internal/payment/infrastructure/postgres/writer"
	"minimart/pkg/outbox"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// EventPublisher writes a domain event into the transactional outbox within the
// caller's transaction, so the event commits atomically with the payment.
// *outbox.Publisher is the production implementation.
type EventPublisher interface {
	Send(ctx context.Context, msg outbox.Message, tx pgx.Tx) error
}

// PaymentRepository is the Postgres adapter for domain.Repository. Reads go
// through the replica pool, writes through the primary.
type PaymentRepository struct {
	writePool *pgxpool.Pool
	reader    *reader.Queries
	writer    *writer.Queries
	events    EventPublisher
}

var _ domain.Repository = (*PaymentRepository)(nil)

// NewPaymentRepository builds the repository over the given pools and outbox
// publisher.
func NewPaymentRepository(readPool, writePool *pgxpool.Pool, events EventPublisher) *PaymentRepository {
	return &PaymentRepository{
		writePool: writePool,
		reader:    reader.New(readPool),
		writer:    writer.New(writePool),
		events:    events,
	}
}

// Save inserts a pending payment and its events in one transaction. ON CONFLICT
// DO NOTHING makes the insert idempotent: if the idempotency key already exists,
// no row is written and ErrIdempotencyConflict is reported so the caller can
// replay the existing payment.
func (r *PaymentRepository) Save(ctx context.Context, p domain.Payment) error {
	tx, err := r.writePool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }() // no-op after a successful Commit

	q := r.writer.WithTx(tx)
	rows, err := q.CreatePayment(ctx, writer.CreatePaymentParams{
		ID:             p.ID,
		OrderID:        p.OrderID,
		UserID:         p.UserID,
		IdempotencyKey: p.IdempotencyKey,
		AmountCents:    p.AmountCents,
		Currency:       p.Currency,
		Method:         p.Method,
		Status:         string(p.Status),
		Provider:       p.Provider,
	})
	if err != nil {
		return err
	}
	if rows == 0 {
		return domain.ErrIdempotencyConflict
	}
	if err := r.appendEvents(ctx, tx, p.Events()); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

// GetByID reads a payment by its id.
func (r *PaymentRepository) GetByID(ctx context.Context, id string) (domain.Payment, error) {
	row, err := r.reader.GetPayment(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.Payment{}, domain.ErrPaymentNotFound
		}
		return domain.Payment{}, err
	}
	return toDomain(row), nil
}

// GetByIdempotencyKey reads a payment by its idempotency key. It is the
// authoritative fallback for idempotency when the Redis cache is missing or
// unavailable.
func (r *PaymentRepository) GetByIdempotencyKey(ctx context.Context, key string) (domain.Payment, error) {
	row, err := r.reader.GetPaymentByIdempotencyKey(ctx, key)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.Payment{}, domain.ErrPaymentNotFound
		}
		return domain.Payment{}, err
	}
	return toDomain(row), nil
}

// Update finalizes a pending payment and writes its events in one transaction.
// The UPDATE is a compare-and-set on status = 'pending', so a payment that is
// already terminal is never reprocessed; that case reports ErrPaymentNotPending.
func (r *PaymentRepository) Update(ctx context.Context, p domain.Payment) error {
	tx, err := r.writePool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	q := r.writer.WithTx(tx)
	rows, err := q.UpdatePaymentStatus(ctx, writer.UpdatePaymentStatusParams{
		ID:            p.ID,
		Status:        string(p.Status),
		ProviderRef:   p.ProviderRef,
		FailureReason: p.FailureReason,
		UpdatedAt:     p.UpdatedAt,
	})
	if err != nil {
		return err
	}
	if rows == 0 {
		return domain.ErrPaymentNotPending
	}
	if err := r.appendEvents(ctx, tx, p.Events()); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

// ListReconcilable returns pending or unknown payments older than updatedBefore.
func (r *PaymentRepository) ListReconcilable(ctx context.Context, updatedBefore time.Time, limit int) ([]domain.Payment, error) {
	rows, err := r.reader.ListReconcilablePayments(ctx, reader.ListReconcilablePaymentsParams{
		UpdatedAt: updatedBefore,
		Limit:     int32(limit), //nolint:gosec // limit is a small batch size
	})
	if err != nil {
		return nil, err
	}
	out := make([]domain.Payment, 0, len(rows))
	for _, row := range rows {
		out = append(out, toDomain(row))
	}
	return out, nil
}

// Reconcile persists a reconciled terminal state for a payment that is still
// pending or unknown, writing its events in the same transaction.
func (r *PaymentRepository) Reconcile(ctx context.Context, p domain.Payment) error {
	tx, err := r.writePool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	q := r.writer.WithTx(tx)
	rows, err := q.ReconcilePayment(ctx, writer.ReconcilePaymentParams{
		ID:            p.ID,
		Status:        string(p.Status),
		ProviderRef:   p.ProviderRef,
		FailureReason: p.FailureReason,
		UpdatedAt:     p.UpdatedAt,
	})
	if err != nil {
		return err
	}
	if rows == 0 {
		return domain.ErrPaymentNotPending
	}
	if err := r.appendEvents(ctx, tx, p.Events()); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

// Refund persists a refunded payment, writing its events in the same
// transaction. The UPDATE is a compare-and-set on status IN ('pending',
// 'succeeded', 'unknown') - the states the domain's Refund accepts - so a
// payment that was already refunded or failed is never reprocessed; that case
// reports ErrPaymentNotRefundable.
func (r *PaymentRepository) Refund(ctx context.Context, p domain.Payment) error {
	tx, err := r.writePool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	q := r.writer.WithTx(tx)
	rows, err := q.RefundPayment(ctx, writer.RefundPaymentParams{
		ID:            p.ID,
		Status:        string(p.Status),
		ProviderRef:   p.ProviderRef,
		FailureReason: p.FailureReason,
		UpdatedAt:     p.UpdatedAt,
	})
	if err != nil {
		return err
	}
	if rows == 0 {
		return domain.ErrPaymentNotRefundable
	}
	if err := r.appendEvents(ctx, tx, p.Events()); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

// ListSucceededByOrderID returns the captured payments for an order.
func (r *PaymentRepository) ListSucceededByOrderID(ctx context.Context, orderID string) ([]domain.Payment, error) {
	rows, err := r.reader.ListSucceededPaymentsByOrderID(ctx, orderID)
	if err != nil {
		return nil, err
	}
	out := make([]domain.Payment, 0, len(rows))
	for _, row := range rows {
		out = append(out, toDomain(row))
	}
	return out, nil
}

// appendEvents writes each domain event to the outbox inside tx. Writing through
// the shared outbox store (which participates in the same tx) is what makes the
// state change and its events atomic.
func (r *PaymentRepository) appendEvents(ctx context.Context, tx pgx.Tx, events []domain.Event) error {
	if len(events) == 0 {
		return nil
	}
	if r.events == nil {
		return errors.New("payment repository: no event publisher configured")
	}
	for _, ev := range events {
		msg, err := outboxMessage(ev)
		if err != nil {
			return err
		}
		if err := r.events.Send(ctx, msg, tx); err != nil {
			return fmt.Errorf("enqueue payment event: %w", err)
		}
	}
	return nil
}

// toDomain maps a database row to the domain aggregate. Domain events are not
// reconstructed: they belong to the write that produced them.
func toDomain(row reader.Payment) domain.Payment {
	return domain.Payment{
		ID:             row.ID,
		OrderID:        row.OrderID,
		UserID:         row.UserID,
		IdempotencyKey: row.IdempotencyKey,
		AmountCents:    row.AmountCents,
		Currency:       row.Currency,
		Method:         row.Method,
		Status:         domain.Status(row.Status),
		Provider:       row.Provider,
		ProviderRef:    row.ProviderRef,
		FailureReason:  row.FailureReason,
		CreatedAt:      row.CreatedAt,
		UpdatedAt:      row.UpdatedAt,
	}
}
