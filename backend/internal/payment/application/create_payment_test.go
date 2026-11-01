package application

import (
	"errors"
	"minimart/internal/order/presentation/module/client"
	"minimart/internal/payment/domain"
	"minimart/pkg/idempotency"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCreatePayment(t *testing.T) {
	t.Run("Idempotency-Key header is missing", func(t *testing.T) {
		e := newTestEnv(t)
		e.noOrderReader()

		req := validCreatePaymentRequest()
		req.IdempotencyKey = ""

		_, err := e.svc.CreatePayment(t.Context(), req)
		require.Error(t, err)
		require.ErrorIs(t, err, ErrIdempotencyKeyRequired)
	})

	t.Run("Amount is not positive", func(t *testing.T) {
		e := newTestEnv(t)
		e.noOrderReader()

		req := validCreatePaymentRequest()
		req.AmountCents = 0

		_, err := e.svc.CreatePayment(t.Context(), req)
		require.Error(t, err)
		require.ErrorIs(t, err, domain.ErrPaymentAmountInvalid)
	})

	t.Run("Currency is missing", func(t *testing.T) {
		e := newTestEnv(t)
		e.noOrderReader()

		req := validCreatePaymentRequest()
		req.Currency = ""

		_, err := e.svc.CreatePayment(t.Context(), req)
		require.Error(t, err)
		require.ErrorIs(t, err, domain.ErrPaymentCurrencyRequired)
	})

	t.Run("Method is missing", func(t *testing.T) {
		e := newTestEnv(t)
		e.noOrderReader()

		req := validCreatePaymentRequest()
		req.Method = ""

		_, err := e.svc.CreatePayment(t.Context(), req)
		require.Error(t, err)
		require.ErrorIs(t, err, domain.ErrPaymentMethodRequired)
	})

	t.Run("User is missing", func(t *testing.T) {
		e := newTestEnv(t)
		e.noOrderReader()

		req := validCreatePaymentRequest()
		req.UserID = ""

		_, err := e.svc.CreatePayment(t.Context(), req)
		require.Error(t, err)
		require.ErrorIs(t, err, domain.ErrPaymentUserRequired)
	})

	t.Run("Order is not found", func(t *testing.T) {
		e := newTestEnv(t)
		e.orderNotFound()

		_, err := e.svc.CreatePayment(t.Context(), validCreatePaymentRequest())
		require.Error(t, err)
		require.ErrorIs(t, err, ErrOrderNotFound)
	})

	t.Run("Payment user does not match order", func(t *testing.T) {
		e := newTestEnv(t)
		e.useOrder(defaultOrder())

		req := validCreatePaymentRequest()
		req.UserID = "u2"

		_, err := e.svc.CreatePayment(t.Context(), req)
		require.Error(t, err)
		require.ErrorIs(t, err, domain.ErrPaymentUserMismatch)
	})

	t.Run("Payment amount does not match order total", func(t *testing.T) {
		e := newTestEnv(t)
		e.useOrder(defaultOrder())

		req := validCreatePaymentRequest()
		req.AmountCents = 2500

		_, err := e.svc.CreatePayment(t.Context(), req)
		require.Error(t, err)
		require.ErrorIs(t, err, domain.ErrPaymentAmountMismatch)
	})

	t.Run("Payment currency does not match order", func(t *testing.T) {
		e := newTestEnv(t)
		e.useOrder(defaultOrder())

		req := validCreatePaymentRequest()
		req.Currency = "EUR"

		_, err := e.svc.CreatePayment(t.Context(), req)
		require.Error(t, err)
		require.ErrorIs(t, err, domain.ErrPaymentCurrencyMismatch)
	})

	t.Run("Same key reused with different parameters", func(t *testing.T) {
		e := newTestEnv(t)
		e.useOrder(defaultOrder())

		req := validCreatePaymentRequest()
		_, err := e.svc.CreatePayment(t.Context(), req)
		require.NoError(t, err)

		req.AmountCents = 2500
		_, err = e.svc.CreatePayment(t.Context(), req)
		require.Error(t, err)
		require.ErrorIs(t, err, domain.ErrIdempotencyConflict)
	})

	t.Run("Storage failure", func(t *testing.T) {
		e := newTestEnv(t)
		e.useOrder(defaultOrder())
		e.repo = &failingRepository{Repository: e.repo, saveErr: errors.New("disk on fire")}
		e.wire()

		req := validCreatePaymentRequest()
		_, err := e.svc.CreatePayment(t.Context(), req)
		require.ErrorContains(t, err, "disk on fire")

		_, reserved := e.idem.records["k1"]
		assert.False(t, reserved, "a failed request frees its reservation")
	})

	t.Run("Provider times out, so the outcome is unknown", func(t *testing.T) {
		e := newTestEnv(t)
		e.useOrder(client.Order{ID: "o1", UserID: "u1", Status: "placed", TotalCents: 7777, Currency: "USD"})
		e.provider = &stubProvider{auth: domain.Result{Outcome: domain.OutcomeTimeout, Reason: "slow"}}
		e.wire()

		req := validCreatePaymentRequest()
		req.AmountCents = 7777

		got, err := e.svc.CreatePayment(t.Context(), req)
		require.NoError(t, err)

		stored := storedPayment(t, e.repo, got.ID)
		assert.Equal(t, domain.StatusUnknown, stored.Status)
		assert.Equal(t, 1, countEvent[domain.PaymentUnknownEvent](stored.Events()))
		assert.Equal(t, 0, countEvent[domain.PaymentSucceededEvent](stored.Events()))
		assert.Equal(t, 0, e.provider.captured)
	})

	t.Run("Redis is unavailable, so idempotency falls back to the database", func(t *testing.T) {
		e := newTestEnv(t)
		e.useOrder(defaultOrder())
		e.idemStore = failingIdem{}
		e.wire()

		req := validCreatePaymentRequest()
		first, err := e.svc.CreatePayment(t.Context(), req)
		require.NoError(t, err)
		assert.Equal(t, domain.StatusSucceeded, first.Status)

		stored := storedPayment(t, e.repo, first.ID)
		assert.Equal(t, domain.StatusSucceeded, stored.Status)

		second, err := e.svc.CreatePayment(t.Context(), req)
		require.NoError(t, err)
		assert.Equal(t, first.ID, second.ID, "the database replay returns the same payment")
		assert.Equal(t, 1, e.provider.authorized)
	})

	t.Run("A concurrent request is still in progress for the same key", func(t *testing.T) {
		e := newTestEnv(t)
		e.useOrder(defaultOrder())

		fp := requestFingerprint("o1", "u1", "USD", "card", 1500)
		e.idem.records["k1"] = idempotency.Record{Fingerprint: fp}

		_, err := e.svc.CreatePayment(t.Context(), validCreatePaymentRequest())
		require.Error(t, err)
		require.ErrorIs(t, err, ErrPaymentInProgress)
		assert.Equal(t, 0, e.provider.authorized)
	})

	t.Run("Payment is approved and captured, and replays return the same payment", func(t *testing.T) {
		e := newTestEnv(t)
		e.useOrder(defaultOrder())

		req := validCreatePaymentRequest()
		first, err := e.svc.CreatePayment(t.Context(), req)
		require.NoError(t, err)
		assert.Equal(t, domain.StatusSucceeded, first.Status)

		stored := storedPayment(t, e.repo, first.ID)
		assert.Equal(t, domain.StatusSucceeded, stored.Status)
		assert.Equal(t, 0, countEvent[domain.PaymentInitiatedEvent](stored.Events()))
		assert.Equal(t, 1, countEvent[domain.PaymentSucceededEvent](stored.Events()))

		second, err := e.svc.CreatePayment(t.Context(), req)
		require.NoError(t, err)
		assert.Equal(t, first.ID, second.ID)
		assert.Equal(t, 1, e.provider.authorized, "the provider is not called again")
		assert.Equal(t, 1, e.provider.captured)
	})

	t.Run("Provider declines the payment", func(t *testing.T) {
		e := newTestEnv(t)
		e.useOrder(defaultOrder())
		e.provider = &stubProvider{auth: domain.Result{Outcome: domain.OutcomeDeclined, Reason: "no funds"}}
		e.wire()

		got, err := e.svc.CreatePayment(t.Context(), validCreatePaymentRequest())
		require.NoError(t, err)

		stored := storedPayment(t, e.repo, got.ID)
		assert.Equal(t, domain.StatusFailed, stored.Status)
		assert.Equal(t, 1, countEvent[domain.PaymentFailedEvent](stored.Events()))
		assert.Equal(t, 0, e.provider.captured)
	})

	t.Run("Payment is rejected for a cancelled order", func(t *testing.T) {
		e := newTestEnv(t)
		e.useOrder(cancelledOrder())

		_, err := e.svc.CreatePayment(t.Context(), validCreatePaymentRequest())
		require.Error(t, err)
		require.ErrorIs(t, err, ErrOrderNotPayable)

		_, err = e.repo.GetByIdempotencyKey(t.Context(), "k1")
		require.ErrorIs(t, err, domain.ErrPaymentNotFound)
	})

	t.Run("Payment is rejected for a failed order", func(t *testing.T) {
		e := newTestEnv(t)
		order := defaultOrder()
		order.Status = "payment_failed"
		e.useOrder(order)

		_, err := e.svc.CreatePayment(t.Context(), validCreatePaymentRequest())
		require.Error(t, err)
		require.ErrorIs(t, err, ErrOrderNotPayable)
	})

	t.Run("Order is cancelled while the charge is in flight", func(t *testing.T) {
		e := newTestEnv(t)
		e.reader = &stubOrderReader{order: defaultOrder(), flipAfter: 1, flipStatus: "cancelled"}
		e.wire()

		got, err := e.svc.CreatePayment(t.Context(), validCreatePaymentRequest())
		require.NoError(t, err)

		stored := storedPayment(t, e.repo, got.ID)
		assert.Equal(t, domain.StatusRefunded, stored.Status)
		require.Equal(t, 1, countEvent[domain.PaymentRefundedEvent](stored.Events()))
		for _, ev := range stored.Events() {
			if refunded, ok := ev.(domain.PaymentRefundedEvent); ok {
				assert.Equal(t, "pedido cancelado", refunded.Reason)
			}
		}
		assert.Equal(t, 1, e.provider.captured)
		assert.Equal(t, 1, e.provider.refunded)
	})

	t.Run("Order is cancelled while the charge is in flight and the refund fails", func(t *testing.T) {
		e := newTestEnv(t)
		e.reader = &stubOrderReader{order: defaultOrder(), flipAfter: 1, flipStatus: "cancelled"}
		e.provider.refundErr = errors.New("gateway down")
		e.wire()

		_, err := e.svc.CreatePayment(t.Context(), validCreatePaymentRequest())
		require.Error(t, err)
		require.ErrorIs(t, err, ErrRefundFailed)

		stored, err := e.repo.GetByIdempotencyKey(t.Context(), "k1")
		require.NoError(t, err)
		assert.Equal(t, domain.StatusPending, stored.Status)
		require.NotNil(t, stored.ProviderRef)
		assert.Equal(t, "ref-1", *stored.ProviderRef)
	})

	t.Run("Provider settles but the response is lost, so the payment is reconciled", func(t *testing.T) {
		e := newTestEnv(t)
		e.useOrder(client.Order{ID: "o1", UserID: "u1", Status: "placed", TotalCents: 8888, Currency: "USD"})
		e.provider = &stubProvider{
			auth:    domain.Result{Outcome: domain.OutcomeApproved, ProviderRef: "ref-1"},
			capture: domain.Result{Outcome: domain.OutcomeTimeout, ProviderRef: "ref-1", Reason: "answer lost"},
		}
		e.wire()

		req := validCreatePaymentRequest()
		req.AmountCents = 8888
		got, err := e.svc.CreatePayment(t.Context(), req)
		require.NoError(t, err)

		stored := storedPayment(t, e.repo, got.ID)
		assert.Equal(t, domain.StatusUnknown, stored.Status)
		require.NotNil(t, stored.ProviderRef)
		assert.Equal(t, "ref-1", *stored.ProviderRef)
		assert.Equal(t, 1, countEvent[domain.PaymentUnknownEvent](stored.Events()))

		reconcileOnce(t, e.svc)

		stored = storedPayment(t, e.repo, got.ID)
		assert.Equal(t, domain.StatusSucceeded, stored.Status)
		assert.Equal(t, 1, countEvent[domain.PaymentSucceededEvent](stored.Events()))
		assert.Equal(t, 1, e.provider.captured)
	})

	t.Run("Capture succeeds but final persistence fails, so the charge reference is kept", func(t *testing.T) {
		e := newTestEnv(t)
		e.useOrder(defaultOrder())
		e.repo = &failingRepository{Repository: e.repo, updateErr: errors.New("disk on fire")}
		e.wire()

		_, err := e.svc.CreatePayment(t.Context(), validCreatePaymentRequest())
		require.ErrorContains(t, err, "disk on fire")

		stored, err := e.repo.GetByIdempotencyKey(t.Context(), "k1")
		require.NoError(t, err)
		assert.Equal(t, domain.StatusPending, stored.Status)
		require.NotNil(t, stored.ProviderRef)
		assert.Equal(t, "ref-1", *stored.ProviderRef)

		reconcileOnce(t, e.svc)

		stored = storedPayment(t, e.repo, stored.ID)
		assert.Equal(t, domain.StatusSucceeded, stored.Status)
		assert.Equal(t, 1, countEvent[domain.PaymentSucceededEvent](stored.Events()))
	})

	t.Run("Two concurrent requests with the same key create a single payment", func(t *testing.T) {
		e := newTestEnv(t)
		e.useOrder(defaultOrder())
		e.idemStore = failingIdem{}
		e.wire()

		entered := make(chan struct{})
		release := make(chan struct{})
		var once sync.Once
		e.provider.authorizeHook = func() {
			once.Do(func() { close(entered) })
			<-release
		}

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
				p, err := e.svc.CreatePayment(t.Context(), validCreatePaymentRequest())
				results[i] = result{payment: p, err: err}
				done <- i
			}(i)
		}

		<-entered
		<-done
		close(release)
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
			case errors.Is(r.err, ErrPaymentInProgress):
				inProgress++
			default:
				require.NoError(t, r.err)
			}
		}
		assert.GreaterOrEqual(t, successes, 1)
		assert.LessOrEqual(t, inProgress, 1)

		stored, err := e.repo.GetByIdempotencyKey(t.Context(), "k1")
		require.NoError(t, err)
		assert.Equal(t, successID, stored.ID)
		assert.Equal(t, domain.StatusSucceeded, stored.Status)
		assert.Equal(t, 1, e.provider.authorized)
		assert.Equal(t, 1, e.provider.captured)
	})
}
