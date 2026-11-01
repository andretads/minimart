package application

import (
	"context"
	"errors"
	"minimart/internal/order/domain"
	"minimart/pkg/metrics"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type autoCancelRepo struct {
	orders      []domain.Order
	listErr     error
	updateErr   error
	updateCalls int
}

func (r *autoCancelRepo) Save(_ context.Context, o domain.Order) error { return nil }
func (r *autoCancelRepo) GetByID(_ context.Context, _ string) (domain.Order, error) {
	return domain.Order{}, domain.ErrOrderNotFound
}

func (r *autoCancelRepo) GetByIdempotencyKey(_ context.Context, _ string) (domain.Order, error) {
	return domain.Order{}, domain.ErrOrderNotFound
}

func (r *autoCancelRepo) Update(_ context.Context, _ domain.Order, _ domain.Status) error {
	r.updateCalls++
	return r.updateErr
}

func (r *autoCancelRepo) ListByUserID(_ context.Context, _ string, _, _ int) ([]domain.Order, error) {
	return nil, nil
}
func (r *autoCancelRepo) CountByUserID(_ context.Context, _ string) (int, error) { return 0, nil }
func (r *autoCancelRepo) ListPlacedBefore(_ context.Context, _ time.Time, _ int) ([]domain.Order, error) {
	if r.listErr != nil {
		return nil, r.listErr
	}
	return r.orders, nil
}

func TestAutoCancellerRunOnce(t *testing.T) {
	t.Run("cancels every stale placed order and records success", func(t *testing.T) {
		o2, err := domain.NewOrder("o2", "u1", "c1", "", []domain.Item{
			{ProductID: "p1", Name: "Widget", UnitPriceCents: 500, Quantity: 1},
		})
		require.NoError(t, err)

		repo := &autoCancelRepo{orders: []domain.Order{placedOrder(t), o2}}
		m := &countingMetrics{}
		ac := NewAutoCanceller(repo, m, 10, nil)

		ac.RunOnce(t.Context())

		assert.Equal(t, 2, repo.updateCalls)
		assert.Equal(t, []string{metrics.StatusSuccess, metrics.StatusSuccess}, m.cancelled)
	})

	t.Run("a list failure skips the run", func(t *testing.T) {
		boom := errors.New("disk on fire")
		repo := &autoCancelRepo{listErr: boom}
		m := &countingMetrics{}
		ac := NewAutoCanceller(repo, m, 10, nil)

		ac.RunOnce(t.Context())

		assert.Equal(t, 0, repo.updateCalls)
		assert.Empty(t, m.cancelled)
	})

	t.Run("a lost status race is skipped without a metric", func(t *testing.T) {
		repo := &autoCancelRepo{orders: []domain.Order{placedOrder(t)}, updateErr: domain.ErrOrderStateChanged}
		m := &countingMetrics{}
		ac := NewAutoCanceller(repo, m, 10, nil)

		ac.RunOnce(t.Context())

		assert.Equal(t, 1, repo.updateCalls)
		assert.Empty(t, m.cancelled)
	})

	t.Run("a persist failure is skipped without a metric", func(t *testing.T) {
		boom := errors.New("disk on fire")
		repo := &autoCancelRepo{orders: []domain.Order{placedOrder(t)}, updateErr: boom}
		m := &countingMetrics{}
		ac := NewAutoCanceller(repo, m, 10, nil)

		ac.RunOnce(t.Context())

		assert.Equal(t, 1, repo.updateCalls)
		assert.Empty(t, m.cancelled)
	})

	t.Run("an order that can no longer be cancelled is skipped", func(t *testing.T) {
		paid := placedOrder(t).MarkPaid()
		repo := &autoCancelRepo{orders: []domain.Order{paid}}
		m := &countingMetrics{}
		ac := NewAutoCanceller(repo, m, 10, nil)

		ac.RunOnce(t.Context())

		assert.Equal(t, 0, repo.updateCalls)
		assert.Empty(t, m.cancelled)
	})

	t.Run("defaults are applied for non-positive settings", func(t *testing.T) {
		ac := NewAutoCanceller(&autoCancelRepo{}, metrics.Noop{}, 0, nil)
		assert.Equal(t, 100, ac.limit)
		assert.NotNil(t, ac.logger)
	})
}
