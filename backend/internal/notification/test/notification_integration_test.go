//go:build integration

package test

import (
	"minimart/internal/notification/application"
	"minimart/internal/notification/domain"
	"minimart/pkg/metrics"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func uid(t *testing.T, suffix string) string {
	t.Helper()
	return t.Name() + "-" + suffix
}

func TestNotificationApplication_SendOrderConfirmation(t *testing.T) {
	ctx := t.Context()
	repo := newTestRepo(t)
	sender := newFakeEmailSender()
	resolver := newFakeRecipientResolver()
	svc := application.NewService(repo, newTestDeliveryRepo(t), newTestPrefsRepo(t), sender, nil, nil, resolver, nil, metrics.Noop{}, application.RetryConfig{})

	t.Run("creates new confirmation", func(t *testing.T) {
		orderID := uid(t, "order-1")
		userID := uid(t, "user-1")
		resolver.SetEmail(userID, "user1@example.com")
		err := svc.SendOrderConfirmation(ctx, orderID, userID, 1500)
		require.NoError(t, err)

		stored, err := repo.GetByOrderIDAndType(ctx, orderID, domain.TypeOrderConfirmation)
		require.NoError(t, err)
		assert.Equal(t, domain.TypeOrderConfirmation, stored.Type)
		assert.Equal(t, orderID, stored.OrderID)
		assert.Equal(t, userID, stored.UserID)
		assert.Contains(t, stored.Body, "R$ 15,00")
	})

	t.Run("idempotent on duplicate (OrderID, Type)", func(t *testing.T) {
		orderID := uid(t, "order-2")
		userID := uid(t, "user-2")
		resolver.SetEmail(userID, "user2@example.com")

		err := svc.SendOrderConfirmation(ctx, orderID, userID, 2500)
		require.NoError(t, err)

		err = svc.SendOrderConfirmation(ctx, orderID, userID, 2500)
		require.NoError(t, err)

		all, err := repo.ListByUserID(ctx, userID, 1, 20)
		require.NoError(t, err)
		assert.Len(t, all, 1)
		assert.Equal(t, domain.TypeOrderConfirmation, all[0].Type)
	})

	t.Run("different type for same order stores both", func(t *testing.T) {
		orderID := uid(t, "order-3")
		userID := uid(t, "user-3")
		resolver.SetEmail(userID, "user3@example.com")
		err := svc.SendOrderConfirmation(ctx, orderID, userID, 1000)
		require.NoError(t, err)

		err = svc.SendOrderCancellation(ctx, orderID, userID, 1000)
		require.NoError(t, err)

		all, err := repo.ListByUserID(ctx, userID, 1, 20)
		require.NoError(t, err)
		assert.Len(t, all, 2)
	})
}

func TestNotificationApplication_SendOrderCancellation(t *testing.T) {
	ctx := t.Context()
	repo := newTestRepo(t)
	sender := newFakeEmailSender()
	resolver := newFakeRecipientResolver()
	svc := application.NewService(repo, newTestDeliveryRepo(t), newTestPrefsRepo(t), sender, nil, nil, resolver, nil, metrics.Noop{}, application.RetryConfig{})
	resolver.SetEmail(uid(t, "user-cancel"), "user@example.com")

	orderID := uid(t, "order-cancel")
	userID := uid(t, "user-cancel")
	err := svc.SendOrderCancellation(ctx, orderID, userID, 2000)
	require.NoError(t, err)

	stored, err := repo.GetByOrderIDAndType(ctx, orderID, domain.TypeOrderCancellation)
	require.NoError(t, err)
	assert.Equal(t, domain.TypeOrderCancellation, stored.Type)
	assert.Equal(t, orderID, stored.OrderID)
	assert.Contains(t, stored.Body, "R$ 20,00")
}

func TestNotificationApplication_SendPaymentSucceeded(t *testing.T) {
	ctx := t.Context()
	repo := newTestRepo(t)
	sender := newFakeEmailSender()
	resolver := newFakeRecipientResolver()
	svc := application.NewService(repo, newTestDeliveryRepo(t), newTestPrefsRepo(t), sender, nil, nil, resolver, nil, metrics.Noop{}, application.RetryConfig{})
	resolver.SetEmail(uid(t, "user-4"), "user@example.com")

	orderID := uid(t, "order-4")
	userID := uid(t, "user-4")
	err := svc.SendPaymentSucceeded(ctx, orderID, userID, 3000)
	require.NoError(t, err)

	stored, err := repo.GetByOrderIDAndType(ctx, orderID, domain.TypePaymentSucceeded)
	require.NoError(t, err)
	assert.Equal(t, domain.TypePaymentSucceeded, stored.Type)
	assert.Contains(t, stored.Body, "R$ 30,00")
}

func TestNotificationApplication_SendPaymentFailed(t *testing.T) {
	ctx := t.Context()
	repo := newTestRepo(t)
	sender := newFakeEmailSender()
	resolver := newFakeRecipientResolver()
	svc := application.NewService(repo, newTestDeliveryRepo(t), newTestPrefsRepo(t), sender, nil, nil, resolver, nil, metrics.Noop{}, application.RetryConfig{})
	resolver.SetEmail(uid(t, "user-5"), "user@example.com")

	orderID := uid(t, "order-5")
	userID := uid(t, "user-5")
	err := svc.SendPaymentFailed(ctx, orderID, userID, 2500, "insufficient_funds")
	require.NoError(t, err)

	stored, err := repo.GetByOrderIDAndType(ctx, orderID, domain.TypePaymentFailed)
	require.NoError(t, err)
	assert.Equal(t, domain.TypePaymentFailed, stored.Type)
	assert.Contains(t, stored.Body, "R$ 25,00")
}

func TestNotificationApplication_ConcurrentSave(t *testing.T) {
	ctx := t.Context()
	repo := newTestRepo(t)
	sender := newFakeEmailSender()
	resolver := newFakeRecipientResolver()
	svc := application.NewService(repo, newTestDeliveryRepo(t), newTestPrefsRepo(t), sender, nil, nil, resolver, nil, metrics.Noop{}, application.RetryConfig{})
	resolver.SetEmail(uid(t, "user-race"), "race@example.com")

	orderID := uid(t, "order-race")
	userID := uid(t, "user-race")
	const writers = 10
	errs := make(chan error, writers)
	var wg sync.WaitGroup

	for range writers {
		wg.Go(func() {
			errs <- svc.SendOrderConfirmation(ctx, orderID, userID, 1000)
		})
	}
	wg.Wait()
	close(errs)

	for err := range errs {
		require.NoError(t, err)
	}

	all, err := repo.ListByUserID(ctx, userID, 1, 20)
	require.NoError(t, err)
	assert.Len(t, all, 1, "racing saves of one event must store exactly one notification")
}
