package event

import (
	"context"
	"encoding/json"
	"minimart/internal/payment/application"
	"minimart/internal/payment/domain"
	"minimart/internal/payment/infrastructure/memory"
	"minimart/internal/payment/infrastructure/provider/fake"
	"minimart/pkg/idempotency"
	"minimart/pkg/metrics"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type stubIdempotency struct{}

func (stubIdempotency) Get(context.Context, string) (idempotency.Record, bool, error) {
	return idempotency.Record{}, false, nil
}

func (stubIdempotency) Reserve(context.Context, string, string, time.Duration) (bool, error) {
	return true, nil
}

func (stubIdempotency) Complete(context.Context, string, string, string, time.Duration) error {
	return nil
}

func (stubIdempotency) Release(context.Context, string, string) error { return nil }

// newHandlerWithPayment stores a succeeded payment for order o1 and returns a
// handler over a service backed by the same repository.
func newHandlerWithPayment(t *testing.T) (*Handler, *memory.Repository) {
	t.Helper()
	repo := memory.New()
	p, err := domain.NewPayment("pay-1", "o1", "u1", "idem-1", 1500, "USD", "card", "fake")
	require.NoError(t, err)
	p, err = p.ApplyOutcome(domain.Result{Outcome: domain.OutcomeApproved, ProviderRef: "ref-1"})
	require.NoError(t, err)
	require.NoError(t, repo.Save(t.Context(), p))

	svc := application.NewService(repo, nil, stubIdempotency{}, fake.New(fake.Config{}), metrics.Noop{}, time.Hour)
	return NewHandler(svc), repo
}

func TestOrderCancelled(t *testing.T) {
	t.Run("malformed JSON returns an error", func(t *testing.T) {
		h, _ := newHandlerWithPayment(t)
		require.Error(t, h.OrderCancelled(t.Context(), []byte("not json")))
	})

	t.Run("cancelled order refunds its payments", func(t *testing.T) {
		h, repo := newHandlerWithPayment(t)

		msg, err := json.Marshal(map[string]any{"order_id": "o1", "user_id": "u1", "total_cents": 1500})
		require.NoError(t, err)
		require.NoError(t, h.OrderCancelled(t.Context(), msg))

		p, err := repo.GetByID(t.Context(), "pay-1")
		require.NoError(t, err)
		assert.Equal(t, domain.StatusRefunded, p.Status)
	})

	t.Run("order without payments is a no-op", func(t *testing.T) {
		h, repo := newHandlerWithPayment(t)

		msg, err := json.Marshal(map[string]any{"order_id": "no-such-order", "user_id": "u1", "total_cents": 1500})
		require.NoError(t, err)
		require.NoError(t, h.OrderCancelled(t.Context(), msg))

		p, err := repo.GetByID(t.Context(), "pay-1")
		require.NoError(t, err)
		assert.Equal(t, domain.StatusSucceeded, p.Status)
	})

	t.Run("replayed event is idempotent", func(t *testing.T) {
		h, repo := newHandlerWithPayment(t)
		msg, err := json.Marshal(map[string]any{"order_id": "o1", "user_id": "u1", "total_cents": 1500})
		require.NoError(t, err)

		require.NoError(t, h.OrderCancelled(t.Context(), msg))
		require.NoError(t, h.OrderCancelled(t.Context(), msg))

		p, err := repo.GetByID(t.Context(), "pay-1")
		require.NoError(t, err)
		assert.Equal(t, domain.StatusRefunded, p.Status)
	})
}
