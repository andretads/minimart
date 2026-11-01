package application

import (
	"errors"
	catalogclient "minimart/internal/catalog/presentation/module/client"
	"minimart/internal/order/domain"
	"minimart/pkg/metrics"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPlaceOrder(t *testing.T) {
	t.Run("Order has no items", func(t *testing.T) {
		repo := &fakeRepo{}
		svc := newSvc(repo, testCatalog(), metrics.Noop{})

		_, err := svc.PlaceOrder(t.Context(), PlaceOrderRequest{UserID: "u1", IdempotencyKey: "k1"})
		require.ErrorIs(t, err, domain.ErrNoItems)
		assert.Empty(t, repo.saved.ID, "no order is stored")
	})

	t.Run("Item quantity is not positive", func(t *testing.T) {
		repo := &fakeRepo{}
		svc := newSvc(repo, testCatalog(), metrics.Noop{})

		_, err := svc.PlaceOrder(t.Context(), PlaceOrderRequest{
			UserID: "u1", IdempotencyKey: "k1", Items: []PlaceOrderItem{{ProductID: "p1", Quantity: 0}},
		})
		require.ErrorIs(t, err, domain.ErrInvalidQuantity)
		assert.Empty(t, repo.saved.ID, "no order is stored")
	})

	t.Run("Product does not exist", func(t *testing.T) {
		repo := &fakeRepo{}
		svc := newSvc(repo, testCatalog(), metrics.Noop{})

		_, err := svc.PlaceOrder(t.Context(), PlaceOrderRequest{
			UserID: "u1", IdempotencyKey: "k1", Items: []PlaceOrderItem{{ProductID: "nope", Quantity: 1}},
		})
		require.ErrorIs(t, err, catalogclient.ErrProductNotFound)
		assert.ErrorContains(t, err, "resolve product nope:")
	})

	t.Run("Product does not have enough stock", func(t *testing.T) {
		repo := &fakeRepo{}
		modules := testCatalog()
		modules.stock = map[string]int{"p1": 0}
		svc := newSvc(repo, modules, metrics.Noop{})

		_, err := svc.PlaceOrder(t.Context(), validPlaceOrderRequest("u1", "k1"))
		require.ErrorIs(t, err, catalogclient.ErrInsufficientStock)
		assert.Empty(t, repo.saved.ID, "no order is stored")
	})

	t.Run("Missing idempotency key", func(t *testing.T) {
		svc := newSvc(&fakeRepo{}, testCatalog(), metrics.Noop{})

		_, err := svc.PlaceOrder(t.Context(), PlaceOrderRequest{
			UserID: "u1", Items: []PlaceOrderItem{{ProductID: "p1", Quantity: 1}},
		})
		require.ErrorIs(t, err, ErrIdempotencyKeyRequired)
	})

	t.Run("Storage failure releases the reservation", func(t *testing.T) {
		boom := errors.New("disk on fire")
		repo := &fakeRepo{saveErr: boom}
		modules := testCatalog()
		modules.stock = map[string]int{"p1": 5}
		svc := newSvc(repo, modules, metrics.Noop{})

		_, err := svc.PlaceOrder(t.Context(), validPlaceOrderRequest("u1", "k1"))
		require.ErrorIs(t, err, boom)
		require.Len(t, modules.reserved, 1)
		assert.Equal(t, modules.reserved, modules.released, "the stock reservation is released")
	})

	t.Run("Reusing a key with different parameters is a conflict", func(t *testing.T) {
		svc := newSvc(&fakeRepo{}, testCatalog(), metrics.Noop{})

		first := validPlaceOrderRequest("u1", "k1")
		_, err := svc.PlaceOrder(t.Context(), first)
		require.NoError(t, err)

		second := validPlaceOrderRequest("u1", "k1")
		second.Items = []PlaceOrderItem{{ProductID: "p1", Quantity: 2}}
		_, err = svc.PlaceOrder(t.Context(), second)
		require.ErrorIs(t, err, domain.ErrIdempotencyConflict)
	})

	t.Run("A concurrent request is still in progress for the same key", func(t *testing.T) {
		req := validPlaceOrderRequest("u1", "k1")
		idem := newMemIdempotency()
		ok, err := idem.Reserve(t.Context(), req.IdempotencyKey, requestFingerprint(req), time.Minute)
		require.NoError(t, err)
		require.True(t, ok)

		svc := NewService(&fakeRepo{}, testCatalog(), idem, metrics.Noop{}, time.Hour)
		_, err = svc.PlaceOrder(t.Context(), req)
		require.ErrorIs(t, err, ErrOrderInProgress)
	})

	t.Run("Redis is unavailable, so idempotency falls back to the database", func(t *testing.T) {
		repo := &fakeRepo{}
		modules := testCatalog()
		modules.stock = map[string]int{"p1": 5}
		svc := NewService(repo, modules, failingIdempotency{}, metrics.Noop{}, time.Hour)
		req := validPlaceOrderRequest("u1", "k1")

		_, err := svc.PlaceOrder(t.Context(), req)
		require.NoError(t, err)
		assert.Equal(t, "u1", repo.saved.UserID)
		require.Len(t, repo.saved.Items, 1)
		assert.Equal(t, int64(500), repo.saved.TotalCents)

		first := repo.saved
		_, err = svc.PlaceOrder(t.Context(), req)
		require.NoError(t, err)
		assert.Equal(t, first.ID, repo.saved.ID, "the same order id is returned")
	})

	t.Run("Order got placed, stock reserved, event outboxed", func(t *testing.T) {
		repo := &fakeRepo{}
		modules := testCatalog()
		modules.stock = map[string]int{"p1": 5, "p2": 5}
		svc := newSvc(repo, modules, metrics.Noop{})

		got, err := svc.PlaceOrder(t.Context(), PlaceOrderRequest{
			UserID: "u1", IdempotencyKey: "k1",
			Items: []PlaceOrderItem{{ProductID: "p1", Quantity: 2}, {ProductID: "p2", Quantity: 1}},
		})
		require.NoError(t, err)

		assert.Equal(t, "u1", repo.saved.UserID)
		require.Len(t, repo.saved.Items, 2)
		assert.Equal(t, int64(1250), repo.saved.TotalCents)
		assert.Equal(t, []string{got.ID}, modules.reserved, "order stock was reserved for the order")

		event := findPlaced(t, repo.saved)
		assert.Equal(t, got.ID, event.OrderID)
	})

	t.Run("A retry with the same key replays the stored order", func(t *testing.T) {
		repo := &fakeRepo{}
		modules := testCatalog()
		modules.stock = map[string]int{"p1": 5}
		svc := newSvc(repo, modules, metrics.Noop{})
		req := validPlaceOrderRequest("u1", "k1")

		first, err := svc.PlaceOrder(t.Context(), req)
		require.NoError(t, err)
		second, err := svc.PlaceOrder(t.Context(), req)
		require.NoError(t, err)

		assert.Equal(t, first.ID, second.ID, "the same order id is returned")
		assert.Len(t, modules.reserved, 1, "order stock was reserved exactly once")
	})

	t.Run("The shipping snapshot is stored with the order", func(t *testing.T) {
		repo := &fakeRepo{}
		svc := newSvc(repo, testCatalog(), metrics.Noop{})
		req := validPlaceOrderRequest("u1", "k1")
		req.ShippingQuoteID = "q1"
		req.ShippingMethodID = "m1"
		req.ShippingMethodName = "Standard"
		req.ShippingCostCents = 250
		req.DestinationSnapshot = featureDestination()

		_, err := svc.PlaceOrder(t.Context(), req)
		require.NoError(t, err)

		assertShippingSnapshot(t, repo.saved)
		assert.Equal(t, int64(750), repo.saved.TotalCents)
	})

	t.Run("An order placed without shipping carries no shipping snapshot", func(t *testing.T) {
		repo := &fakeRepo{}
		svc := newSvc(repo, testCatalog(), metrics.Noop{})

		_, err := svc.PlaceOrder(t.Context(), validPlaceOrderRequest("u1", "k1"))
		require.NoError(t, err)

		assert.Empty(t, repo.saved.ShippingQuoteID)
		assert.Empty(t, repo.saved.ShippingMethodID)
		assert.Zero(t, repo.saved.ShippingCostCents)
		assert.Equal(t, domain.DestinationSnapshot{}, repo.saved.DestinationSnapshot)
	})

	t.Run("A freshly placed order exposes its auto-cancel deadline", func(t *testing.T) {
		svc := newSvc(&fakeRepo{}, testCatalog(), metrics.Noop{})

		got, err := svc.PlaceOrder(t.Context(), validPlaceOrderRequest("u1", "k1"))
		require.NoError(t, err)
		assert.Equal(t, domain.StatusPlaced, got.Status)
		require.NotNil(t, got.AutoCancelAt(), "a freshly placed order has an auto-cancel deadline")
	})
}
