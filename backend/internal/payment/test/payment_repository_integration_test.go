//go:build integration

// Package test holds the payment domain's integration tests.
package test

import (
	"encoding/json"
	"minimart/internal/payment/domain"
	"minimart/pkg/kafka"
	"minimart/pkg/message"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPaymentRepository_SaveGetByID(t *testing.T) {
	repo, pool := newPaymentRepo(t)
	ctx := t.Context()
	id, key := uniquePaymentKey(), uniquePaymentKey()
	cleanup(t, pool, id, key)

	want, err := domain.NewPayment(id, "it-order", "it-user", key, 1500, "USD", "card", "fake")
	require.NoError(t, err)
	require.NoError(t, repo.Save(ctx, want))

	got, err := repo.GetByID(ctx, id)
	require.NoError(t, err)
	assert.Equal(t, id, got.ID)
	assert.Equal(t, "it-order", got.OrderID)
	assert.Equal(t, int64(1500), got.AmountCents)
	assert.Equal(t, domain.StatusPending, got.Status)
	assert.Nil(t, got.ProviderRef)

	byKey, err := repo.GetByIdempotencyKey(ctx, key)
	require.NoError(t, err)
	assert.Equal(t, id, byKey.ID)
}

func TestPaymentRepository_GetByIDNotFound(t *testing.T) {
	repo, _ := newPaymentRepo(t)

	_, err := repo.GetByID(t.Context(), uniquePaymentKey())
	require.ErrorIs(t, err, domain.ErrPaymentNotFound)

	_, err = repo.GetByIdempotencyKey(t.Context(), uniquePaymentKey())
	require.ErrorIs(t, err, domain.ErrPaymentNotFound)
}

// TestPaymentRepository_SaveDuplicateKeyConflicts proves the unique index on
// idempotency_key is the authoritative idempotency guard: a second insert under
// the same key writes no row and reports a conflict.
func TestPaymentRepository_SaveDuplicateKeyConflicts(t *testing.T) {
	repo, pool := newPaymentRepo(t)
	ctx := t.Context()
	id, key := uniquePaymentKey(), uniquePaymentKey()
	cleanup(t, pool, id, key)

	first, err := domain.NewPayment(id, "it-order", "it-user", key, 1500, "USD", "card", "fake")
	require.NoError(t, err)
	require.NoError(t, repo.Save(ctx, first))

	second, err := domain.NewPayment(uniquePaymentKey(), "it-order", "it-user", key, 1500, "USD", "card", "fake")
	require.NoError(t, err)
	assert.ErrorIs(t, repo.Save(ctx, second), domain.ErrIdempotencyConflict)
}

// TestPaymentRepository_UpdateIsCompareAndSet proves Update only transitions a
// pending payment, and rejects a second finalization.
func TestPaymentRepository_UpdateIsCompareAndSet(t *testing.T) {
	repo, pool := newPaymentRepo(t)
	ctx := t.Context()
	id, key := uniquePaymentKey(), uniquePaymentKey()
	cleanup(t, pool, id, key)

	pending, err := domain.NewPayment(id, "it-order", "it-user", key, 1500, "USD", "card", "fake")
	require.NoError(t, err)
	require.NoError(t, repo.Save(ctx, pending))

	final, err := pending.ApplyOutcome(domain.Result{Outcome: domain.OutcomeApproved, ProviderRef: "ref-1"})
	require.NoError(t, err)
	require.NoError(t, repo.Update(ctx, final))

	got, err := repo.GetByID(ctx, id)
	require.NoError(t, err)
	assert.Equal(t, domain.StatusSucceeded, got.Status)
	require.NotNil(t, got.ProviderRef)
	assert.Equal(t, "ref-1", *got.ProviderRef)

	assert.ErrorIs(t, repo.Update(ctx, final), domain.ErrPaymentNotPending)
}

// TestPaymentRepository_SaveWritesOutboxEvent proves the payment and its
// PaymentInitiated event commit together.
func TestPaymentRepository_SaveWritesOutboxEvent(t *testing.T) {
	repo, pool := newPaymentRepo(t)
	ctx := t.Context()
	id, key := uniquePaymentKey(), uniquePaymentKey()
	cleanup(t, pool, id, key)

	pending, err := domain.NewPayment(id, "it-order", "it-user", key, 1500, "USD", "card", "fake")
	require.NoError(t, err)
	require.NoError(t, repo.Save(ctx, pending))

	var (
		topic string
		body  []byte
		state int
	)
	err = pool.QueryRow(ctx, "SELECT topic, body, state FROM outbox WHERE message_key = $1", id).Scan(&topic, &body, &state)
	require.NoError(t, err)

	assert.Equal(t, kafka.PaymentInitiatedTopic.String(), topic)
	assert.Equal(t, 0, state, "a freshly written event is pending")
	var payload message.PaymentInitiated
	require.NoError(t, json.Unmarshal(body, &payload))
	assert.Equal(t, id, payload.PaymentID)
	assert.Equal(t, int64(1500), payload.AmountCents)
}

// TestPaymentRepository_UpdateWritesOutboxEvent proves that updating a payment
// to succeeded writes the corresponding outbox event.
func TestPaymentRepository_UpdateWritesOutboxEvent(t *testing.T) {
	repo, pool := newPaymentRepo(t)
	ctx := t.Context()
	id, key := uniquePaymentKey(), uniquePaymentKey()
	cleanup(t, pool, id, key)

	pending, err := domain.NewPayment(id, "it-order", "it-user", key, 1500, "USD", "card", "fake")
	require.NoError(t, err)
	require.NoError(t, repo.Save(ctx, pending))

	// Update to succeeded
	succeeded, err := pending.ApplyOutcome(domain.Result{Outcome: domain.OutcomeApproved, ProviderRef: "ref-1"})
	require.NoError(t, err)
	require.NoError(t, repo.Update(ctx, succeeded))

	// Check outbox has PaymentSucceeded event
	var topic string
	err = pool.QueryRow(ctx, "SELECT topic FROM outbox WHERE message_key = $1 AND topic = $2", id, kafka.PaymentSucceededTopic.String()).Scan(&topic)
	require.NoError(t, err)
	assert.Equal(t, kafka.PaymentSucceededTopic.String(), topic)
}

// TestPaymentRepository_UpdateWritesPaymentFailedOutboxEvent proves that
// updating a payment to failed writes the PaymentFailed outbox event.
func TestPaymentRepository_UpdateWritesPaymentFailedOutboxEvent(t *testing.T) {
	repo, pool := newPaymentRepo(t)
	ctx := t.Context()
	id, key := uniquePaymentKey(), uniquePaymentKey()
	cleanup(t, pool, id, key)

	pending, err := domain.NewPayment(id, "it-order", "it-user", key, 1500, "USD", "card", "fake")
	require.NoError(t, err)
	require.NoError(t, repo.Save(ctx, pending))

	// Update to failed (decline)
	failed, err := pending.ApplyOutcome(domain.Result{Outcome: domain.OutcomeDeclined, ProviderRef: "ref-decline"})
	require.NoError(t, err)
	require.NoError(t, repo.Update(ctx, failed))

	// Check outbox has PaymentFailed event
	var topic string
	err = pool.QueryRow(ctx, "SELECT topic FROM outbox WHERE message_key = $1 AND topic = $2", id, kafka.PaymentFailedTopic.String()).Scan(&topic)
	require.NoError(t, err)
	assert.Equal(t, kafka.PaymentFailedTopic.String(), topic)
}

// TestPaymentRepository_UpdateWritesPaymentUnknownOutboxEvent proves that
// updating a payment to unknown (technical error) writes the PaymentUnknown outbox event.
func TestPaymentRepository_UpdateWritesPaymentUnknownOutboxEvent(t *testing.T) {
	repo, pool := newPaymentRepo(t)
	ctx := t.Context()
	id, key := uniquePaymentKey(), uniquePaymentKey()
	cleanup(t, pool, id, key)

	pending, err := domain.NewPayment(id, "it-order", "it-user", key, 1500, "USD", "card", "fake")
	require.NoError(t, err)
	require.NoError(t, repo.Save(ctx, pending))

	// Update to unknown (technical error)
	unknown, err := pending.ApplyOutcome(domain.Result{Outcome: domain.OutcomeTechnicalError, Reason: "technical error"})
	require.NoError(t, err)
	require.NoError(t, repo.Update(ctx, unknown))

	// Check outbox has PaymentUnknown event
	var topic string
	err = pool.QueryRow(ctx, "SELECT topic FROM outbox WHERE message_key = $1 AND topic = $2", id, kafka.PaymentUnknownTopic.String()).Scan(&topic)
	require.NoError(t, err)
	assert.Equal(t, kafka.PaymentUnknownTopic.String(), topic)
}

// TestPaymentRepository_UpdateWritesPaymentTimeoutOutboxEvent proves that
// updating a payment to unknown (timeout) writes the PaymentUnknown outbox event.
func TestPaymentRepository_UpdateWritesPaymentTimeoutOutboxEvent(t *testing.T) {
	repo, pool := newPaymentRepo(t)
	ctx := t.Context()
	id, key := uniquePaymentKey(), uniquePaymentKey()
	cleanup(t, pool, id, key)

	pending, err := domain.NewPayment(id, "it-order", "it-user", key, 1500, "USD", "card", "fake")
	require.NoError(t, err)
	require.NoError(t, repo.Save(ctx, pending))

	// Update to unknown (timeout)
	unknown, err := pending.ApplyOutcome(domain.Result{Outcome: domain.OutcomeTimeout, Reason: "timeout"})
	require.NoError(t, err)
	require.NoError(t, repo.Update(ctx, unknown))

	// Check outbox has PaymentUnknown event
	var topic string
	err = pool.QueryRow(ctx, "SELECT topic FROM outbox WHERE message_key = $1 AND topic = $2", id, kafka.PaymentUnknownTopic.String()).Scan(&topic)
	require.NoError(t, err)
	assert.Equal(t, kafka.PaymentUnknownTopic.String(), topic)
}

// TestPaymentRepository_RefundIsCompareAndSet proves Refund only transitions a
// refundable payment, and rejects refunding one already refunded.
func TestPaymentRepository_RefundIsCompareAndSet(t *testing.T) {
	repo, pool := newPaymentRepo(t)
	ctx := t.Context()
	id, key := uniquePaymentKey(), uniquePaymentKey()
	cleanup(t, pool, id, key)

	pending, err := domain.NewPayment(id, "it-order", "it-user", key, 1500, "USD", "card", "fake")
	require.NoError(t, err)
	require.NoError(t, repo.Save(ctx, pending))
	succeeded, err := pending.ApplyOutcome(domain.Result{Outcome: domain.OutcomeApproved, ProviderRef: "ref-1"})
	require.NoError(t, err)
	require.NoError(t, repo.Update(ctx, succeeded))

	refunded, err := succeeded.Refund(domain.Result{Outcome: domain.OutcomeApproved, ProviderRef: "ref-1", Reason: "pedido cancelado"})
	require.NoError(t, err)
	require.NoError(t, repo.Refund(ctx, refunded))

	got, err := repo.GetByID(ctx, id)
	require.NoError(t, err)
	assert.Equal(t, domain.StatusRefunded, got.Status)
	assert.Nil(t, got.FailureReason)

	assert.ErrorIs(t, repo.Refund(ctx, refunded), domain.ErrPaymentNotRefundable)
}

// TestPaymentRepository_RefundWritesOutboxEvent proves a refund commits its
// PaymentRefunded event to the outbox in the same transaction.
func TestPaymentRepository_RefundWritesOutboxEvent(t *testing.T) {
	repo, pool := newPaymentRepo(t)
	ctx := t.Context()
	id, key := uniquePaymentKey(), uniquePaymentKey()
	cleanup(t, pool, id, key)

	pending, err := domain.NewPayment(id, "it-order", "it-user", key, 1500, "USD", "card", "fake")
	require.NoError(t, err)
	require.NoError(t, repo.Save(ctx, pending))
	succeeded, err := pending.ApplyOutcome(domain.Result{Outcome: domain.OutcomeApproved, ProviderRef: "ref-1"})
	require.NoError(t, err)
	require.NoError(t, repo.Update(ctx, succeeded))
	refunded, err := succeeded.Refund(domain.Result{Outcome: domain.OutcomeApproved, ProviderRef: "ref-1", Reason: "pedido cancelado"})
	require.NoError(t, err)
	require.NoError(t, repo.Refund(ctx, refunded))

	var (
		topic string
		body  []byte
	)
	err = pool.QueryRow(ctx, "SELECT topic, body FROM outbox WHERE message_key = $1 AND topic = $2", id, kafka.PaymentRefundedTopic.String()).Scan(&topic, &body)
	require.NoError(t, err)
	assert.Equal(t, kafka.PaymentRefundedTopic.String(), topic)

	var payload message.PaymentRefunded
	require.NoError(t, json.Unmarshal(body, &payload))
	assert.Equal(t, id, payload.PaymentID)
	assert.Equal(t, "it-order", payload.OrderID)
	assert.Equal(t, int64(1500), payload.AmountCents)
	assert.Equal(t, "ref-1", payload.ProviderRef)
	assert.Equal(t, "pedido cancelado", payload.Reason)
}

// TestPaymentRepository_ListSucceededByOrderID proves only captured payments of
// the given order are listed.
func TestPaymentRepository_ListSucceededByOrderID(t *testing.T) {
	repo, pool := newPaymentRepo(t)
	ctx := t.Context()
	orderID := "it-order-" + uniquePaymentKey()

	capturedID, failedID, otherID := uniquePaymentKey(), uniquePaymentKey(), uniquePaymentKey()
	cleanup(t, pool, capturedID, capturedID)
	cleanup(t, pool, failedID, failedID)
	cleanup(t, pool, otherID, otherID)

	mkPayment := func(id, orderID string, outcome domain.Outcome) domain.Payment {
		t.Helper()
		pending, err := domain.NewPayment(id, orderID, "it-user", id, 1500, "USD", "card", "fake")
		require.NoError(t, err)
		require.NoError(t, repo.Save(ctx, pending))
		p, err := pending.ApplyOutcome(domain.Result{Outcome: outcome, ProviderRef: "ref-" + id})
		require.NoError(t, err)
		require.NoError(t, repo.Update(ctx, p))
		return p
	}

	mkPayment(capturedID, orderID, domain.OutcomeApproved)
	mkPayment(failedID, orderID, domain.OutcomeDeclined)
	mkPayment(otherID, "another-order", domain.OutcomeApproved)

	payments, err := repo.ListSucceededByOrderID(ctx, orderID)
	require.NoError(t, err)
	require.Len(t, payments, 1)
	assert.Equal(t, capturedID, payments[0].ID)
	assert.Equal(t, domain.StatusSucceeded, payments[0].Status)
}
