//go:build integration

package test

import (
	"minimart/internal/notification/application"
	"minimart/internal/notification/domain"
	"minimart/pkg/metrics"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUpdatePreferencesIntegration(t *testing.T) {
	ctx := t.Context()
	repo := newTestRepo(t)
	deliveryRepo := newTestDeliveryRepo(t)
	prefsRepo := newTestPrefsRepo(t)
	svc := application.NewService(repo, deliveryRepo, prefsRepo, newFakeEmailSender(), nil, nil, newFakeRecipientResolver(), nil, metrics.Noop{}, application.RetryConfig{})

	t.Run("Empty user_id is rejected", func(t *testing.T) {
		_, err := svc.UpdatePreferences(ctx, application.UpdatePreferencesRequest{UserID: ""})
		require.ErrorIs(t, err, domain.ErrEmptyUserID)
	})

	t.Run("All four switches are replaced", func(t *testing.T) {
		userID := uid(t, "u1")
		got, err := svc.UpdatePreferences(ctx, application.UpdatePreferencesRequest{
			UserID: userID, EmailEnabled: false, SMSEnabled: false, PushEnabled: true, MarketingEnabled: true,
		})
		require.NoError(t, err)
		assert.False(t, got.EmailEnabled)
		assert.False(t, got.SMSEnabled)
		assert.True(t, got.PushEnabled)
		assert.True(t, got.MarketingEnabled)
	})

	t.Run("A later update replaces the previous switches", func(t *testing.T) {
		userID := uid(t, "u2")
		_, err := svc.UpdatePreferences(ctx, application.UpdatePreferencesRequest{
			UserID: userID, EmailEnabled: true, SMSEnabled: true, PushEnabled: true, MarketingEnabled: true,
		})
		require.NoError(t, err)
		got, err := svc.UpdatePreferences(ctx, application.UpdatePreferencesRequest{
			UserID: userID, EmailEnabled: false, SMSEnabled: false, PushEnabled: false, MarketingEnabled: false,
		})
		require.NoError(t, err)
		assert.False(t, got.EmailEnabled)
		assert.False(t, got.SMSEnabled)
		assert.False(t, got.PushEnabled)
		assert.False(t, got.MarketingEnabled)
	})
}
