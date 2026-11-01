//go:build integration

package test

import (
	"encoding/json"
	"errors"
	"minimart/internal/order/presentation/module/client"
	"minimart/internal/payment/application"
	"minimart/internal/payment/domain"
	"minimart/internal/payment/infrastructure/provider/fake"
	"minimart/pkg/idempotency"
	"minimart/pkg/kafka"
	"minimart/pkg/message"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCreatePaymentIntegration(t *testing.T) {
	t.Run("Idempotency-Key header is missing", func(t *testing.T) {
		e := newPaymentEnv(t)
		e.reader = nil
		e.wire()

		req := e.request()
		req.IdempotencyKey = ""

		_, err := e.svc.CreatePayment(t.Context(), req)
		require.Error(t, err)
		require.ErrorIs(t, err, application.ErrIdempotencyKeyRequired)
	})

	t.Run("Amount is not positive", func(t *testing.T) {
		e := newPaymentEnv(t)
		e.reader = nil
		e.wire()

		req := e.request()
		req.AmountCents = 0

		_, err := e.svc.CreatePayment(t.Context(), req)
		require.Error(t, err)
		require.ErrorIs(t, err, domain.ErrPaymentAmountInvalid)
	})

	t.Run("Currency is missing", func(t *testing.T) {
		e := newPaymentEnv(t)
		e.reader = nil
		e.wire()

		req := e.request()
		req.Currency = ""

		_, err := e.svc.CreatePayment(t.Context(), req)
		require.Error(t, err)
		require.ErrorIs(t, err, domain.ErrPaymentCurrencyRequired)
	})

	t.Run("Method is missing", func(t *testing.T) {
		e := newPaymentEnv(t)
		e.reader = nil
		e.wire()

		req := e.request()
		req.Method = ""

		_, err := e.svc.CreatePayment(t.Context(), req)
		require.Error(t, err)
		require.ErrorIs(t, err, domain.ErrPaymentMethodRequired)
	})

	t.Run("User is missing", func(t *testing.T) {
		e := newPaymentEnv(t)
		e.reader = nil
		e.wire()

		req := e.request()
		req.UserID = ""

		_, err := e.svc.CreatePayment(t.Context(), req)
		require.Error(t, err)
		require.ErrorIs(t, err, domain.ErrPaymentUserRequired)
	})

	t.Run("Order is not found", func(t *testing.T) {
		e := newPaymentEnv(t)
		e.reader = &mutableOrderReader{err: client.ErrOrderNotFound}
		e.wire()

		_, err := e.svc.CreatePayment(t.Context(), e.request())
		require.Error(t, err)
		require.ErrorIs(t, err, application.ErrOrderNotFound)
	})

	t.Run("Payment user does not match order", func(t *testing.T) {
		e := newPaymentEnv(t)

		req := e.request()
		req.UserID = e.userID + "-other"

		_, err := e.svc.CreatePayment(t.Context(), req)
		require.Error(t, err)
		require.ErrorIs(t, err, domain.ErrPaymentUserMismatch)
	})

	t.Run("Payment amount does not match order total", func(t *testing.T) {
		e := newPaymentEnv(t)

		req := e.request()
		req.AmountCents = 2500

		_, err := e.svc.CreatePayment(t.Context(), req)
		require.Error(t, err)
		require.ErrorIs(t, err, domain.ErrPaymentAmountMismatch)
	})

	t.Run("Payment currency does not match order", func(t *testing.T) {
		e := newPaymentEnv(t)

		req := e.request()
		req.Currency = "EUR"

		_, err := e.svc.CreatePayment(t.Context(), req)
		require.Error(t, err)
		require.ErrorIs(t, err, domain.ErrPaymentCurrencyMismatch)
	})

	t.Run("Same key reused with different parameters", func(t *testing.T) {
		e := newPaymentEnv(t)

		req := e.request()
		e.createPayment(t, req)

		req.AmountCents = 2500
		_, err := e.svc.CreatePayment(t.Context(), req)
		require.Error(t, err)
		require.ErrorIs(t, err, domain.ErrIdempotencyConflict)
	})

	t.Run("Storage failure", func(t *testing.T) {
		e := newPaymentEnv(t)
		e.repo = failingSaveRepo{Repository: e.pgRepo, err: errors.New("disk on fire")}
		e.wire()

		req := e.request()
		_, err := e.svc.CreatePayment(t.Context(), req)
		require.ErrorContains(t, err, "disk on fire")

		_, found, err := e.idem.Get(t.Context(), req.IdempotencyKey)
		require.NoError(t, err)
		assert.False(t, found, "a failed request frees its reservation")

		_, err = e.pgRepo.GetByIdempotencyKey(t.Context(), req.IdempotencyKey)
		require.ErrorIs(t, err, domain.ErrPaymentNotFound)
	})

	t.Run("Provider times out, so the outcome is unknown", func(t *testing.T) {
		e := newPaymentEnv(t)
		e.reader.order.TotalCents = 7777

		req := e.request()
		req.AmountCents = 7777
		got := e.createPayment(t, req)

		stored := e.stored(t, got.ID)
		assert.Equal(t, domain.StatusUnknown, stored.Status)

		rows := outboxRows(t, e.pool, got.ID)
		assert.Equal(t, 1, countTopic(rows, kafka.PaymentInitiatedTopic))
		assert.Equal(t, 1, countTopic(rows, kafka.PaymentUnknownTopic))
		assert.Equal(t, 0, countTopic(rows, kafka.PaymentSucceededTopic))
	})

	t.Run("Redis is unavailable, so idempotency falls back to the database", func(t *testing.T) {
		e := newPaymentEnv(t)
		e.idem = failingIdem{}
		e.wire()

		req := e.request()
		first := e.createPayment(t, req)
		assert.Equal(t, domain.StatusSucceeded, first.Status)
		assert.Equal(t, domain.StatusSucceeded, e.stored(t, first.ID).Status)

		second, err := e.svc.CreatePayment(t.Context(), req)
		require.NoError(t, err)
		assert.Equal(t, first.ID, second.ID)

		rows := outboxRows(t, e.pool, first.ID)
		assert.Equal(t, 1, countTopic(rows, kafka.PaymentSucceededTopic))
	})

	t.Run("A concurrent request is still in progress for the same key", func(t *testing.T) {
		e := newPaymentEnv(t)

		req := e.request()
		fp := idempotency.Fingerprint(e.orderID, e.userID, "USD", "card", "1500")
		ok, err := e.idem.Reserve(t.Context(), req.IdempotencyKey, fp, time.Minute)
		require.NoError(t, err)
		require.True(t, ok)

		_, err = e.svc.CreatePayment(t.Context(), req)
		require.Error(t, err)
		require.ErrorIs(t, err, application.ErrPaymentInProgress)

		_, err = e.pgRepo.GetByIdempotencyKey(t.Context(), req.IdempotencyKey)
		require.ErrorIs(t, err, domain.ErrPaymentNotFound)
	})

	t.Run("Payment is approved and captured, and replays return the same payment", func(t *testing.T) {
		e := newPaymentEnv(t)

		req := e.request()
		first := e.createPayment(t, req)
		assert.Equal(t, domain.StatusSucceeded, first.Status)
		assert.Equal(t, domain.StatusSucceeded, e.stored(t, first.ID).Status)

		rows := outboxRows(t, e.pool, first.ID)
		assert.Equal(t, 1, countTopic(rows, kafka.PaymentInitiatedTopic))
		assert.Equal(t, 1, countTopic(rows, kafka.PaymentSucceededTopic))

		second, err := e.svc.CreatePayment(t.Context(), req)
		require.NoError(t, err)
		assert.Equal(t, first.ID, second.ID)

		replayRows := outboxRows(t, e.pool, first.ID)
		assert.Equal(t, 1, countTopic(replayRows, kafka.PaymentInitiatedTopic))
		assert.Equal(t, 1, countTopic(replayRows, kafka.PaymentSucceededTopic))
	})

	t.Run("Provider declines the payment", func(t *testing.T) {
		e := newPaymentEnv(t)
		e.reader.order.TotalCents = 9999

		req := e.request()
		req.AmountCents = 9999
		got := e.createPayment(t, req)

		assert.Equal(t, domain.StatusFailed, e.stored(t, got.ID).Status)
		rows := outboxRows(t, e.pool, got.ID)
		assert.Equal(t, 1, countTopic(rows, kafka.PaymentFailedTopic))
		assert.Equal(t, 0, countTopic(rows, kafka.PaymentSucceededTopic))
	})

	t.Run("Payment is rejected for a cancelled order", func(t *testing.T) {
		e := newPaymentEnv(t)
		e.setOrderStatus("cancelled")

		req := e.request()
		_, err := e.svc.CreatePayment(t.Context(), req)
		require.Error(t, err)
		require.ErrorIs(t, err, application.ErrOrderNotPayable)

		_, err = e.pgRepo.GetByIdempotencyKey(t.Context(), req.IdempotencyKey)
		require.ErrorIs(t, err, domain.ErrPaymentNotFound)
	})

	t.Run("Payment is rejected for a failed order", func(t *testing.T) {
		e := newPaymentEnv(t)
		e.setOrderStatus("payment_failed")

		_, err := e.svc.CreatePayment(t.Context(), e.request())
		require.Error(t, err)
		require.ErrorIs(t, err, application.ErrOrderNotPayable)
	})

	t.Run("Order is cancelled while the charge is in flight", func(t *testing.T) {
		e := newPaymentEnv(t)
		e.reader.flipAfter = 1
		e.reader.flipStatus = "cancelled"

		got := e.createPayment(t, e.request())

		assert.Equal(t, domain.StatusRefunded, e.stored(t, got.ID).Status)
		rows := outboxRows(t, e.pool, got.ID)
		assert.Equal(t, 1, countTopic(rows, kafka.PaymentRefundedTopic))
		assert.Equal(t, 0, countTopic(rows, kafka.PaymentSucceededTopic))

		var refunded message.PaymentRefunded
		for _, row := range rows {
			if row.Topic == kafka.PaymentRefundedTopic.String() {
				require.NoError(t, json.Unmarshal(row.Body, &refunded))
			}
		}
		assert.Equal(t, "pedido cancelado", refunded.Reason)
	})

	t.Run("Order is cancelled while the charge is in flight and the refund fails", func(t *testing.T) {
		e := newPaymentEnv(t)
		e.reader.flipAfter = 1
		e.reader.flipStatus = "cancelled"
		e.provider = failingRefundProvider{Provider: fake.New(fake.Config{}), err: errors.New("gateway down")}
		e.wire()

		req := e.request()
		_, err := e.svc.CreatePayment(t.Context(), req)
		require.Error(t, err)
		require.ErrorIs(t, err, application.ErrRefundFailed)

		stored, err := e.pgRepo.GetByIdempotencyKey(t.Context(), req.IdempotencyKey)
		require.NoError(t, err)
		cleanup(t, e.pool, stored.ID, req.IdempotencyKey)
		assert.Equal(t, domain.StatusPending, stored.Status)
		require.NotNil(t, stored.ProviderRef)
		assert.NotEmpty(t, *stored.ProviderRef)

		rows := outboxRows(t, e.pool, stored.ID)
		assert.Equal(t, 0, countTopic(rows, kafka.PaymentRefundedTopic))
	})

	t.Run("Provider settles but the response is lost, so the payment is reconciled", func(t *testing.T) {
		e := newPaymentEnv(t)
		e.reader.order.TotalCents = 8888

		req := e.request()
		req.AmountCents = 8888
		got := e.createPayment(t, req)

		stored := e.stored(t, got.ID)
		assert.Equal(t, domain.StatusUnknown, stored.Status)
		require.NotNil(t, stored.ProviderRef)
		assert.NotEmpty(t, *stored.ProviderRef)

		rows := outboxRows(t, e.pool, got.ID)
		assert.Equal(t, 1, countTopic(rows, kafka.PaymentUnknownTopic))
		assert.Equal(t, 0, countTopic(rows, kafka.PaymentSucceededTopic))

		e.reconcile(t)

		assert.Equal(t, domain.StatusSucceeded, e.stored(t, got.ID).Status)
		rows = outboxRows(t, e.pool, got.ID)
		assert.Equal(t, 1, countTopic(rows, kafka.PaymentSucceededTopic))
	})

	t.Run("Capture succeeds but final persistence fails, so the charge reference is kept", func(t *testing.T) {
		e := newPaymentEnv(t)
		e.repo = failingUpdateRepo{Repository: e.pgRepo, err: errors.New("disk on fire")}
		e.wire()

		req := e.request()
		_, err := e.svc.CreatePayment(t.Context(), req)
		require.ErrorContains(t, err, "disk on fire")

		stored, err := e.pgRepo.GetByIdempotencyKey(t.Context(), req.IdempotencyKey)
		require.NoError(t, err)
		cleanup(t, e.pool, stored.ID, req.IdempotencyKey)
		assert.Equal(t, domain.StatusPending, stored.Status)
		require.NotNil(t, stored.ProviderRef)
		assert.NotEmpty(t, *stored.ProviderRef)

		e.reconcile(t)

		assert.Equal(t, domain.StatusSucceeded, e.stored(t, stored.ID).Status)
		rows := outboxRows(t, e.pool, stored.ID)
		assert.Equal(t, 1, countTopic(rows, kafka.PaymentSucceededTopic))
	})

	t.Run("Two concurrent requests with the same key create a single payment", func(t *testing.T) {
		e := newPaymentEnv(t)
		e.idem = failingIdem{}
		fakeProvider := fake.New(fake.Config{})
		blocker := newBlockingProvider(fakeProvider)
		e.provider = blocker
		e.wire()

		req := e.request()
		type result struct {
			payment domain.Payment
			err     error
		}
		results := make([]result, 2)
		done := make(chan int, 2)
		var wg sync.WaitGroup
		for i := range 2 {
			wg.Add(1)
			go func(i int) {
				defer wg.Done()
				p, err := e.svc.CreatePayment(t.Context(), req)
				results[i] = result{payment: p, err: err}
				done <- i
			}(i)
		}

		<-blocker.entered
		<-done
		close(blocker.release)
		wg.Wait()

		var successID string
		var successes, inProgress int
		for _, r := range results {
			switch {
			case r.err == nil:
				successes++
				assert.NotEmpty(t, r.payment.ID)
				if successID == "" {
					successID = r.payment.ID
				} else {
					assert.Equal(t, successID, r.payment.ID)
				}
			case errors.Is(r.err, application.ErrPaymentInProgress):
				inProgress++
			default:
				require.NoError(t, r.err)
			}
		}
		assert.GreaterOrEqual(t, successes, 1)
		assert.LessOrEqual(t, inProgress, 1)

		var count int
		var storedID string
		require.NoError(t, e.pool.QueryRow(t.Context(),
			"SELECT COUNT(*), MIN(id) FROM payments WHERE idempotency_key = $1", req.IdempotencyKey).Scan(&count, &storedID))
		assert.Equal(t, 1, count)
		assert.Equal(t, successID, storedID)
		assert.Equal(t, 1, fakeProvider.CaptureCalls(req.IdempotencyKey))
		cleanup(t, e.pool, storedID, req.IdempotencyKey)
	})
}
