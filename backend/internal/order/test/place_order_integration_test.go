//go:build integration

package test

import (
	"encoding/json"
	"errors"
	catalogclient "minimart/internal/catalog/presentation/module/client"
	"minimart/internal/order/application"
	"minimart/internal/order/domain"
	"minimart/pkg/kafka"
	"minimart/pkg/message"
	"minimart/pkg/metrics"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// validPlaceRequest is the canonical placement payload: one Widget.
func validPlaceRequest(userID, idempotencyKey string) application.PlaceOrderRequest {
	return application.PlaceOrderRequest{
		UserID:         userID,
		IdempotencyKey: idempotencyKey,
		Items:          []application.PlaceOrderItem{{ProductID: "p1", Quantity: 1}},
	}
}

func TestPlaceOrderIntegration(t *testing.T) {
	t.Run("Order has no items", func(t *testing.T) {
		svc, _, pool, _ := newOrderService(t)
		userID := uniqueUser(t, pool)

		_, err := svc.PlaceOrder(t.Context(), application.PlaceOrderRequest{
			UserID: userID, IdempotencyKey: uniqueOrderID(),
		})
		require.ErrorIs(t, err, domain.ErrNoItems)
	})

	t.Run("Item quantity is not positive", func(t *testing.T) {
		svc, _, pool, _ := newOrderService(t)
		userID := uniqueUser(t, pool)

		_, err := svc.PlaceOrder(t.Context(), application.PlaceOrderRequest{
			UserID: userID, IdempotencyKey: uniqueOrderID(),
			Items: []application.PlaceOrderItem{{ProductID: "p1", Quantity: 0}},
		})
		require.ErrorIs(t, err, domain.ErrInvalidQuantity)
	})

	t.Run("Product does not exist", func(t *testing.T) {
		svc, _, pool, _ := newOrderService(t)
		userID := uniqueUser(t, pool)

		_, err := svc.PlaceOrder(t.Context(), application.PlaceOrderRequest{
			UserID: userID, IdempotencyKey: uniqueOrderID(),
			Items: []application.PlaceOrderItem{{ProductID: "nope", Quantity: 1}},
		})
		require.ErrorIs(t, err, catalogclient.ErrProductNotFound)
		assert.ErrorContains(t, err, "resolve product nope:")
	})

	t.Run("Product does not have enough stock", func(t *testing.T) {
		svc, repo, pool, modules := newOrderService(t)
		userID := uniqueUser(t, pool)
		modules.stock["p1"] = 0

		_, err := svc.PlaceOrder(t.Context(), validPlaceRequest(userID, uniqueOrderID()))
		require.ErrorIs(t, err, catalogclient.ErrInsufficientStock)

		count, err := repo.CountByUserID(t.Context(), userID)
		require.NoError(t, err)
		assert.Zero(t, count, "no order is stored")
	})

	t.Run("Missing idempotency key", func(t *testing.T) {
		svc, _, pool, _ := newOrderService(t)
		userID := uniqueUser(t, pool)

		_, err := svc.PlaceOrder(t.Context(), application.PlaceOrderRequest{
			UserID: userID, Items: []application.PlaceOrderItem{{ProductID: "p1", Quantity: 1}},
		})
		require.ErrorIs(t, err, application.ErrIdempotencyKeyRequired)
	})

	t.Run("Storage failure releases the reservation", func(t *testing.T) {
		repo, pool := newOrderRepo(t)
		userID := uniqueUser(t, pool)
		modules := newFakeModules()
		boom := errors.New("disk on fire")
		svc := application.NewService(
			failingSaveRepo{OrderRepository: repo, err: boom}, modules, newMemIdem(), metrics.Noop{}, time.Hour,
		)

		_, err := svc.PlaceOrder(t.Context(), validPlaceRequest(userID, uniqueOrderID()))
		require.ErrorIs(t, err, boom)
		require.Len(t, modules.reserved, 1)
		assert.Equal(t, modules.reserved, modules.released, "the stock reservation is released")

		count, err := repo.CountByUserID(t.Context(), userID)
		require.NoError(t, err)
		assert.Zero(t, count, "no order is stored")
	})

	t.Run("Reusing a key with different parameters is a conflict", func(t *testing.T) {
		svc, repo, pool, _ := newOrderService(t)
		userID := uniqueUser(t, pool)
		key := uniqueOrderID()

		order, err := svc.PlaceOrder(t.Context(), validPlaceRequest(userID, key))
		require.NoError(t, err)
		cleanup(t, pool, order.ID)

		found, err := repo.GetByIdempotencyKey(t.Context(), key)
		require.NoError(t, err)
		assert.Equal(t, order.ID, found.ID)

		_, err = repo.GetByIdempotencyKey(t.Context(), uniqueOrderID())
		require.ErrorIs(t, err, domain.ErrOrderNotFound)

		changed := validPlaceRequest(userID, key)
		changed.Items = []application.PlaceOrderItem{{ProductID: "p1", Quantity: 2}}
		_, err = svc.PlaceOrder(t.Context(), changed)
		require.ErrorIs(t, err, domain.ErrIdempotencyConflict)

		duplicate, err := domain.NewOrder(uniqueOrderID(), userID, "", key, []domain.Item{
			{ProductID: "p1", Name: "Widget", UnitPriceCents: 500, Quantity: 2},
		})
		require.NoError(t, err)
		cleanup(t, pool, duplicate.ID)
		require.ErrorIs(t, repo.Save(t.Context(), duplicate), domain.ErrIdempotencyConflict,
			"the database index refuses a second order with the same key")
	})

	t.Run("A concurrent request is still in progress for the same key", func(t *testing.T) {
		idem := newMemIdem()
		svc, _, pool, _ := newOrderServiceWithIdem(t, idem)
		userID := uniqueUser(t, pool)
		req := validPlaceRequest(userID, uniqueOrderID())

		ok, err := idem.Reserve(t.Context(), req.IdempotencyKey, placeOrderFingerprint(req), time.Minute)
		require.NoError(t, err)
		require.True(t, ok)

		_, err = svc.PlaceOrder(t.Context(), req)
		require.ErrorIs(t, err, application.ErrOrderInProgress)
	})

	t.Run("Redis is unavailable, so idempotency falls back to the database", func(t *testing.T) {
		svc, repo, pool, _ := newOrderServiceWithIdem(t, unavailableIdem{})
		userID := uniqueUser(t, pool)
		req := validPlaceRequest(userID, uniqueOrderID())

		first, err := svc.PlaceOrder(t.Context(), req)
		require.NoError(t, err)
		cleanup(t, pool, first.ID)

		stored, err := repo.GetByID(t.Context(), first.ID)
		require.NoError(t, err)
		assert.Equal(t, userID, stored.UserID)
		require.Len(t, stored.Items, 1)
		assert.Equal(t, int64(500), stored.TotalCents)

		second, err := svc.PlaceOrder(t.Context(), req)
		require.NoError(t, err)
		assert.Equal(t, first.ID, second.ID, "the same order id as before")
	})

	t.Run("Order got placed, stock reserved, event outboxed", func(t *testing.T) {
		svc, repo, pool, modules := newOrderService(t)
		userID := uniqueUser(t, pool)
		key := uniqueOrderID()

		order, err := svc.PlaceOrder(t.Context(), application.PlaceOrderRequest{
			UserID: userID, IdempotencyKey: key,
			Items: []application.PlaceOrderItem{{ProductID: "p1", Quantity: 2}, {ProductID: "p2", Quantity: 1}},
		})
		require.NoError(t, err)
		cleanup(t, pool, order.ID)

		stored, err := repo.GetByID(t.Context(), order.ID)
		require.NoError(t, err)
		assert.Equal(t, userID, stored.UserID)
		require.Len(t, stored.Items, 2)
		assert.Equal(t, int64(1250), stored.TotalCents)
		assert.Equal(t, []string{stored.ID}, modules.reserved, "order stock was reserved for the order")

		byKey, err := repo.GetByIdempotencyKey(t.Context(), key)
		require.NoError(t, err)
		assert.Equal(t, stored.ID, byKey.ID)

		assert.Equal(t, 1, countOutboxTopic(t.Context(), t, pool, stored.ID, kafka.OrderPlacedTopic.String()))
		var (
			messageKey string
			body       []byte
			state      int
		)
		require.NoError(t, pool.QueryRow(t.Context(),
			"SELECT message_key, body, state FROM outbox WHERE message_key = $1 AND topic = $2",
			stored.ID, kafka.OrderPlacedTopic.String(),
		).Scan(&messageKey, &body, &state))
		assert.Equal(t, stored.ID, messageKey)
		assert.Zero(t, state, "a freshly written event is pending")

		var event message.OrderPlaced
		require.NoError(t, json.Unmarshal(body, &event))
		assert.Equal(t, stored.ID, event.OrderID)
		assert.Equal(t, int64(1250), event.TotalCents)
		assert.Equal(t, 2, event.ItemCount)
	})

	t.Run("A retry with the same key replays the stored order", func(t *testing.T) {
		svc, _, pool, modules := newOrderService(t)
		userID := uniqueUser(t, pool)
		req := validPlaceRequest(userID, uniqueOrderID())

		first, err := svc.PlaceOrder(t.Context(), req)
		require.NoError(t, err)
		cleanup(t, pool, first.ID)
		second, err := svc.PlaceOrder(t.Context(), req)
		require.NoError(t, err)

		assert.Equal(t, first.ID, second.ID, "the same order id is returned")
		assert.Len(t, modules.reserved, 1, "order stock was reserved exactly once")
	})

	t.Run("The shipping snapshot is stored with the order", func(t *testing.T) {
		svc, repo, pool, _ := newOrderService(t)
		userID := uniqueUser(t, pool)
		req := validPlaceRequest(userID, uniqueOrderID())
		req.ShippingQuoteID = "q1"
		req.ShippingMethodID = "m1"
		req.ShippingMethodName = "Standard"
		req.ShippingCostCents = 250
		req.DestinationSnapshot = featureDestination()

		order, err := svc.PlaceOrder(t.Context(), req)
		require.NoError(t, err)
		cleanup(t, pool, order.ID)

		stored, err := repo.GetByID(t.Context(), order.ID)
		require.NoError(t, err)
		assert.Equal(t, "q1", stored.ShippingQuoteID)
		assert.Equal(t, "m1", stored.ShippingMethodID)
		assert.Equal(t, int64(250), stored.ShippingCostCents)
		assert.Equal(t, featureDestination(), stored.DestinationSnapshot)
		assert.Equal(t, int64(750), stored.TotalCents)
	})

	t.Run("An order placed without shipping carries no shipping snapshot", func(t *testing.T) {
		svc, repo, pool, _ := newOrderService(t)
		userID := uniqueUser(t, pool)

		order, err := svc.PlaceOrder(t.Context(), validPlaceRequest(userID, uniqueOrderID()))
		require.NoError(t, err)
		cleanup(t, pool, order.ID)

		stored, err := repo.GetByID(t.Context(), order.ID)
		require.NoError(t, err)
		assert.Empty(t, stored.ShippingQuoteID)
		assert.Zero(t, stored.ShippingCostCents)
	})

	t.Run("A freshly placed order exposes its auto-cancel deadline", func(t *testing.T) {
		svc, _, pool, _ := newOrderService(t)
		userID := uniqueUser(t, pool)

		order, err := svc.PlaceOrder(t.Context(), validPlaceRequest(userID, uniqueOrderID()))
		require.NoError(t, err)
		cleanup(t, pool, order.ID)

		assert.Equal(t, domain.StatusPlaced, order.Status)
		require.NotNil(t, order.AutoCancelAt(), "a freshly placed order has an auto-cancel deadline")
	})
}
