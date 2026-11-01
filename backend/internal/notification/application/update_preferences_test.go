package application

import (
	"errors"
	"minimart/internal/notification/domain"
	"minimart/internal/notification/infrastructure/memory"
	"minimart/pkg/metrics"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUpdatePreferences(t *testing.T) {
	t.Run("Empty user_id is rejected", func(t *testing.T) {
		repo := memory.New()
		svc := NewService(repo, repo, repo, nil, nil, nil, nil, nil, metrics.Noop{}, RetryConfig{})

		_, err := svc.UpdatePreferences(t.Context(), UpdatePreferencesRequest{
			UserID: "", EmailEnabled: false, SMSEnabled: false, PushEnabled: true, MarketingEnabled: true,
		})
		require.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrEmptyUserID)
	})

	t.Run("Preferences storage failure is propagated", func(t *testing.T) {
		boom := errors.New("mongo down")
		svc := NewService(memory.New(), memory.New(), failingPrefsRepo{err: boom}, nil, nil, nil, nil, nil, metrics.Noop{}, RetryConfig{})

		_, err := svc.UpdatePreferences(t.Context(), UpdatePreferencesRequest{
			UserID: "u1", EmailEnabled: false, SMSEnabled: false, PushEnabled: true, MarketingEnabled: true,
		})
		require.Error(t, err)
		assert.ErrorIs(t, err, boom)
	})

	t.Run("All four switches are replaced", func(t *testing.T) {
		repo := memory.New()
		svc := NewService(repo, repo, repo, nil, nil, nil, nil, nil, metrics.Noop{}, RetryConfig{})

		got, err := svc.UpdatePreferences(t.Context(), UpdatePreferencesRequest{
			UserID: "u1", EmailEnabled: false, SMSEnabled: false, PushEnabled: true, MarketingEnabled: true,
		})
		require.NoError(t, err)
		assert.False(t, got.EmailEnabled)
		assert.False(t, got.SMSEnabled)
		assert.True(t, got.PushEnabled)
		assert.True(t, got.MarketingEnabled)
		assert.False(t, got.UpdatedAt.IsZero())
	})

	t.Run("A later update replaces the previous switches", func(t *testing.T) {
		repo := memory.New()
		svc := NewService(repo, repo, repo, nil, nil, nil, nil, nil, metrics.Noop{}, RetryConfig{})

		_, err := svc.UpdatePreferences(t.Context(), UpdatePreferencesRequest{
			UserID: "u1", EmailEnabled: true, SMSEnabled: true, PushEnabled: true, MarketingEnabled: true,
		})
		require.NoError(t, err)

		got, err := svc.UpdatePreferences(t.Context(), UpdatePreferencesRequest{
			UserID: "u1", EmailEnabled: false, SMSEnabled: false, PushEnabled: false, MarketingEnabled: false,
		})
		require.NoError(t, err)
		assert.False(t, got.EmailEnabled)
		assert.False(t, got.SMSEnabled)
		assert.False(t, got.PushEnabled)
		assert.False(t, got.MarketingEnabled)
	})
}
