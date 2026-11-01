package application

import (
	"context"
	"errors"
	"log/slog"
	"minimart/internal/payment/domain"
	"minimart/pkg/idempotency"
	"strconv"

	orderclient "minimart/internal/order/presentation/module/client"
)

// CreatePaymentRequest is the input to CreatePayment. IdempotencyKey is the
// caller-supplied key that makes retries safe.
type CreatePaymentRequest struct {
	OrderID        string
	UserID         string
	IdempotencyKey string
	AmountCents    int64
	Currency       string
	Method         string
}

// CreatePayment creates a payment and processes it with the provider,
// idempotently. A repeated call with the same key and parameters replays the
// stored payment; the same key with different parameters is a conflict.
//
// Redis is a fast cache in front of the database: if it is unavailable the flow
// falls back to the payments table's unique index, which is the authoritative
// guard against duplicates.
func (s *Service) CreatePayment(ctx context.Context, req CreatePaymentRequest) (domain.Payment, error) {
	if req.IdempotencyKey == "" {
		return domain.Payment{}, ErrIdempotencyKeyRequired
	}
	key := req.IdempotencyKey
	fp := requestFingerprint(req.OrderID, req.UserID, req.Currency, req.Method, req.AmountCents)

	// Fast path: has this key already produced a payment?
	p, done, err := s.existing(ctx, key, fp)
	if err != nil {
		return domain.Payment{}, err
	}
	if done {
		return p, nil
	}

	// Load and verify the order before charging: payment must not trust the
	// caller's order_id, user_id, amount_cents, or currency.
	if s.orderReader != nil {
		ord, orderErr := s.orderReader.GetOrder(ctx, req.OrderID)
		if orderErr != nil {
			if errors.Is(orderErr, orderclient.ErrOrderNotFound) {
				return domain.Payment{}, ErrOrderNotFound
			}
			return domain.Payment{}, orderErr
		}
		if ord.UserID != req.UserID {
			return domain.Payment{}, ErrPaymentUserMismatch
		}
		if ord.TotalCents != req.AmountCents {
			return domain.Payment{}, ErrPaymentAmountMismatch
		}
		if ord.Currency != "" && ord.Currency != req.Currency {
			return domain.Payment{}, ErrPaymentCurrencyMismatch
		}
		if ord.Status == "cancelled" || ord.Status == "payment_failed" {
			return domain.Payment{}, ErrOrderNotPayable
		}
	}

	// Claim the key so a concurrent duplicate fails fast instead of double
	// charging. On a store error this fails open (returns true).
	if !s.reserve(ctx, key, fp) {
		p, done, err = s.existing(ctx, key, fp)
		if err != nil {
			return domain.Payment{}, err
		}
		if done {
			return p, nil
		}
		return domain.Payment{}, ErrPaymentInProgress
	}

	payment, err := s.process(ctx, req)
	if err != nil {
		// A different request won the race and created this key's payment. Look
		// it up in the database, not the cache: our own reservation is the only
		// Redis entry at this point.
		if errors.Is(err, domain.ErrIdempotencyConflict) {
			p, done, lookupErr := s.lookupByKey(ctx, key, fp)
			if lookupErr != nil {
				return domain.Payment{}, lookupErr
			}
			if done {
				if p.Status == domain.StatusPending {
					return domain.Payment{}, ErrPaymentInProgress
				}
				return p, nil
			}
		}
		// Free the reservation so the caller may retry.
		s.release(ctx, key, fp)
		return domain.Payment{}, err
	}

	s.complete(ctx, key, fp, payment.ID)
	return payment, nil
}

// process builds, saves, charges, and finalizes a single payment.
func (s *Service) process(ctx context.Context, req CreatePaymentRequest) (domain.Payment, error) {
	pending, err := domain.NewPayment(newID(), req.OrderID, req.UserID, req.IdempotencyKey, req.AmountCents, req.Currency, req.Method, s.provider.Name())
	if err != nil {
		return domain.Payment{}, err
	}
	if err = s.repo.Save(ctx, pending); err != nil {
		return domain.Payment{}, err
	}

	result := s.charge(ctx, pending)
	if s.chargeLandsOnCancelledOrder(ctx, req.OrderID, result) {
		captured := pending
		captured.ProviderRef = &result.ProviderRef
		refunded, refundErr := s.refund(ctx, captured, refundReasonOrderCancelled)
		if refundErr != nil {
			// The charge is captured on a cancelled order but could not be
			// reversed now. Persist the charge reference on the still-pending
			// payment so the reconciler can retry the refund; fail the request
			// instead of finalizing it.
			if updateErr := s.repo.Update(ctx, captured); updateErr != nil {
				return domain.Payment{}, errors.Join(refundErr, updateErr)
			}
			return domain.Payment{}, refundErr
		}
		s.m.ObservePaymentAmount(float64(refunded.AmountCents))
		return refunded, nil
	}

	final, err := pending.ApplyOutcome(result)
	if err != nil {
		return domain.Payment{}, err
	}
	if err := s.repo.Update(ctx, final); err != nil {
		return domain.Payment{}, err
	}

	s.m.IncPayment(string(final.Status))
	s.m.ObservePaymentAmount(float64(final.AmountCents))
	return final, nil
}

// chargeLandsOnCancelledOrder reports whether a just-approved charge should be
// reversed because its order was cancelled while the charge was in flight.
// When the order status cannot be read it fails open: the payment finalizes
// normally and the reconciler settles the refund later.
func (s *Service) chargeLandsOnCancelledOrder(ctx context.Context, orderID string, result domain.Result) bool {
	if result.Outcome != domain.OutcomeApproved {
		return false
	}
	cancelled, err := s.orderCancelled(ctx, orderID)
	if err != nil {
		slog.WarnContext(ctx, "payment: order status unavailable after capture; the reconciler settles the refund", "order_id", orderID, "error", err)
		return false
	}
	return cancelled
}

// charge authorizes and, on approval, captures the payment. A provider error of
// any kind maps to a technical error, which the domain records as unknown - never
// as failed, because the charge's fate is unknown.
func (s *Service) charge(ctx context.Context, p domain.Payment) domain.Result {
	auth, err := s.provider.Authorize(ctx, domain.AuthorizationRequest{
		PaymentID:   p.ID,
		OrderID:     p.OrderID,
		AmountCents: p.AmountCents,
		Currency:    p.Currency,
		Method:      p.Method,
	})
	if err != nil {
		return domain.Result{Outcome: domain.OutcomeTechnicalError, Reason: err.Error()}
	}
	if auth.Outcome != domain.OutcomeApproved {
		return auth
	}
	capture, err := s.provider.Capture(ctx, auth.ProviderRef, p.AmountCents)
	if err != nil {
		return domain.Result{Outcome: domain.OutcomeTechnicalError, Reason: err.Error()}
	}
	return capture
}

// existing resolves an idempotency key to an already-known payment. It returns
// found=false with a nil error when the key is unknown and the caller may
// proceed. A store error fails open: the lookup falls back to the database.
func (s *Service) existing(ctx context.Context, key, fp string) (domain.Payment, bool, error) {
	return s.idemCoord.Existing(ctx, key, fp)
}

func (s *Service) lookupByKey(ctx context.Context, key, fp string) (domain.Payment, bool, error) {
	p, err := s.repo.GetByIdempotencyKey(ctx, key)
	if errors.Is(err, domain.ErrPaymentNotFound) {
		return domain.Payment{}, false, nil
	}
	if err != nil {
		return domain.Payment{}, false, err
	}
	if requestFingerprint(p.OrderID, p.UserID, p.Currency, p.Method, p.AmountCents) != fp {
		return domain.Payment{}, false, domain.ErrIdempotencyConflict
	}
	return p, true, nil
}

// reserve claims the idempotency key. It fails open: on a store error it returns
// true so processing continues, relying on the database's unique index.
func (s *Service) reserve(ctx context.Context, key, fp string) bool {
	return s.idemCoord.Reserve(ctx, key, fp)
}

func (s *Service) complete(ctx context.Context, key, fp, paymentID string) {
	s.idemCoord.Complete(ctx, key, fp, paymentID)
}

func (s *Service) release(ctx context.Context, key, fp string) {
	s.idemCoord.Release(ctx, key, fp)
}

// requestFingerprint derives the idempotency fingerprint from the fields that
// define a payment request. Reusing a key with any of these changed is a conflict.
func requestFingerprint(orderID, userID, currency, method string, amountCents int64) string {
	return idempotency.Fingerprint(orderID, userID, currency, method, strconv.FormatInt(amountCents, 10))
}
