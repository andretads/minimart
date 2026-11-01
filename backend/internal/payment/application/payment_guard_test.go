package application

import (
	"context"
	"minimart/internal/order/presentation/module/client"
	"minimart/internal/payment/domain"
	"minimart/pkg/metrics"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type stubOrderReader struct {
	order client.Order
	err   error
}

func (s *stubOrderReader) GetOrder(_ context.Context, _ string) (client.Order, error) {
	if s.err != nil {
		return client.Order{}, s.err
	}
	return s.order, nil
}

func TestCreatePaymentOrderNotPayable(t *testing.T) {
	for _, status := range []string{"cancelled", "payment_failed"} {
		t.Run(status, func(t *testing.T) {
			repo := newFakeRepo()
			idem := newFakeIdem()
			provider := &stubProvider{name: "fake", auth: domain.Result{Outcome: domain.OutcomeApproved}, capture: domain.Result{Outcome: domain.OutcomeApproved}}
			reader := &stubOrderReader{order: client.Order{
				ID:         "o1",
				UserID:     "u1",
				Status:     status,
				TotalCents: 1500,
				Currency:   "USD",
			}}
			svc := NewService(repo, reader, idem, provider, metrics.Noop{}, time.Hour)

			_, err := svc.CreatePayment(t.Context(), CreatePaymentRequest{
				OrderID:        "o1",
				UserID:         "u1",
				IdempotencyKey: "key-" + status,
				AmountCents:    1500,
				Currency:       "USD",
				Method:         "card",
			})
			require.ErrorIs(t, err, ErrOrderNotPayable)
			assert.Empty(t, repo.byID)
		})
	}
}

func TestCreatePaymentOrderNotFoundThroughReader(t *testing.T) {
	repo := newFakeRepo()
	idem := newFakeIdem()
	provider := &stubProvider{name: "fake"}
	reader := &stubOrderReader{err: client.ErrOrderNotFound}
	svc := NewService(repo, reader, idem, provider, metrics.Noop{}, time.Hour)

	_, err := svc.CreatePayment(t.Context(), CreatePaymentRequest{
		OrderID:        "missing",
		UserID:         "u1",
		IdempotencyKey: "key",
		AmountCents:    1500,
		Currency:       "USD",
		Method:         "card",
	})
	require.ErrorIs(t, err, ErrOrderNotFound)
}

func TestCreatePaymentMismatchedStatusStillValidated(t *testing.T) {
	repo := newFakeRepo()
	idem := newFakeIdem()
	provider := &stubProvider{name: "fake", auth: domain.Result{Outcome: domain.OutcomeApproved}, capture: domain.Result{Outcome: domain.OutcomeApproved}}
	reader := &stubOrderReader{order: client.Order{
		ID:         "o1",
		UserID:     "u2",
		Status:     "placed",
		TotalCents: 1500,
		Currency:   "USD",
	}}
	svc := NewService(repo, reader, idem, provider, metrics.Noop{}, time.Hour)

	_, err := svc.CreatePayment(t.Context(), CreatePaymentRequest{
		OrderID:        "o1",
		UserID:         "u1",
		IdempotencyKey: "key",
		AmountCents:    1500,
		Currency:       "USD",
		Method:         "card",
	})
	require.ErrorIs(t, err, ErrPaymentUserMismatch)
}
