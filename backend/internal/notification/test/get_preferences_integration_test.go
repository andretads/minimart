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

func TestGetPreferencesIntegration(t *testing.T) {
	ctx := t.Context()
	repo := newTestRepo(t)
	deliveryRepo := newTestDeliveryRepo(t)
	prefsRepo := newTestPrefsRepo(t)
	svc := application.NewService(repo, deliveryRepo, prefsRepo, newFakeEmailSender(), nil, nil, newFakeRecipientResolver(), nil, metrics.Noop{}, application.RetryConfig{})

	t.Run("Empty user_id is rejected", func(t *testing.T) {
		_, err := svc.GetPreferences(ctx, "")
		require.ErrorIs(t, err, domain.ErrEmptyUserID)
	})

	t.Run("A user with no saved preferences gets the defaults", func(t *testing.T) {
		got, err := svc.GetPreferences(ctx, uid(t, "u1"))
		require.NoError(t, err)
		assert.True(t, got.EmailEnabled)
		assert.False(t, got.SMSEnabled)
		assert.False(t, got.PushEnabled)
		assert.False(t, got.MarketingEnabled)
		assert.True(t, got.UpdatedAt.IsZero())
	})

	t.Run("Saved preferences are returned unchanged", func(t *testing.T) {
		userID := uid(t, "u1b")
		stored, err := domain.NewPreferences(userID, false, true, false, true)
		require.NoError(t, err)
		require.NoError(t, prefsRepo.Upsert(ctx, stored))
		got, err := svc.GetPreferences(ctx, userID)
		require.NoError(t, err)
		assert.False(t, got.EmailEnabled)
		assert.True(t, got.SMSEnabled)
		assert.False(t, got.PushEnabled)
		assert.True(t, got.MarketingEnabled)
		assert.False(t, got.UpdatedAt.IsZero())
	})
}
