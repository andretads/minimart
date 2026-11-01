//go:build integration

package test

import (
	"minimart/internal/notification/application"
	"minimart/pkg/metrics"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSendPaymentFailedIntegration(t *testing.T) {
	ctx := t.Context()
	repo := newTestRepo(t)
	deliveryRepo := newTestDeliveryRepo(t)
	prefsRepo := newTestPrefsRepo(t)
	resolver := newFakeRecipientResolver()
	svc := application.NewService(repo, deliveryRepo, prefsRepo, newFakeEmailSender(), nil, nil, resolver, nil, metrics.Noop{}, application.RetryConfig{})

	t.Run("Empty order id is rejected", func(t *testing.T) {
		err := svc.SendPaymentFailed(ctx, "", uid(t, "u1"), 1250, "reason")
		require.Error(t, err)
	})
}
