//go:build integration

package test

import (
	"minimart/internal/notification/application"
	"minimart/pkg/metrics"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSendOrderConfirmationIntegration(t *testing.T) {
	ctx := t.Context()
	repo := newTestRepo(t)
	deliveryRepo := newTestDeliveryRepo(t)
	prefsRepo := newTestPrefsRepo(t)
	resolver := newFakeRecipientResolver()
	svc := application.NewService(repo, deliveryRepo, prefsRepo, newFakeEmailSender(), nil, nil, resolver, nil, metrics.Noop{}, application.RetryConfig{})

	t.Run("Empty order id is rejected", func(t *testing.T) {
		err := svc.SendOrderConfirmation(ctx, "", uid(t, "u1"), 1250)
		require.Error(t, err)
	})

	t.Run("Empty user_id is rejected", func(t *testing.T) {
		err := svc.SendOrderConfirmation(ctx, uid(t, "o1"), "", 1250)
		require.Error(t, err)
	})

	t.Run("Non-positive total is rejected", func(t *testing.T) {
		err := svc.SendOrderConfirmation(ctx, uid(t, "o1"), uid(t, "u1"), 0)
		require.Error(t, err)
	})
}
