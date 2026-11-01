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
	t.Run("stores the full set of switches", func(t *testing.T) {
		repo := memory.New()
		m := &countingMetrics{}
		svc := NewService(repo, repo, repo, nil, nil, nil, nil, nil, m, RetryConfig{})

		got, err := svc.UpdatePreferences(t.Context(), UpdatePreferencesRequest{
			UserID:           "u-1",
			EmailEnabled:     false,
			SMSEnabled:       true,
			PushEnabled:      false,
			MarketingEnabled: true,
		})
		require.NoError(t, err)
		assert.False(t, got.EmailEnabled)
		assert.True(t, got.SMSEnabled)
		assert.False(t, got.PushEnabled)
		assert.True(t, got.MarketingEnabled)
		assert.False(t, got.UpdatedAt.IsZero())
		assert.Equal(t, []string{metrics.StatusSuccess}, m.prefsUpdate)

		stored, err := repo.GetByUserID(t.Context(), "u-1")
		require.NoError(t, err)
		assert.Equal(t, got, stored)
	})

	t.Run("a later update replaces the previous one", func(t *testing.T) {
		repo := memory.New()
		svc := NewService(repo, repo, repo, nil, nil, nil, nil, nil, metrics.Noop{}, RetryConfig{})

		_, err := svc.UpdatePreferences(t.Context(), UpdatePreferencesRequest{UserID: "u-1", MarketingEnabled: true})
		require.NoError(t, err)
		_, err = svc.UpdatePreferences(t.Context(), UpdatePreferencesRequest{UserID: "u-1"})
		require.NoError(t, err)

		stored, err := repo.GetByUserID(t.Context(), "u-1")
		require.NoError(t, err)
		assert.False(t, stored.MarketingEnabled)
	})

	t.Run("empty user id is rejected", func(t *testing.T) {
		repo := memory.New()
		m := &countingMetrics{}
		svc := NewService(repo, repo, repo, nil, nil, nil, nil, nil, m, RetryConfig{})

		_, err := svc.UpdatePreferences(t.Context(), UpdatePreferencesRequest{})
		require.ErrorIs(t, err, domain.ErrEmptyUserID)
		assert.Equal(t, []string{metrics.StatusError}, m.prefsUpdate)
	})

	t.Run("repository error is propagated", func(t *testing.T) {
		boom := errors.New("mongo down")
		m := &countingMetrics{}
		svc := NewService(memory.New(), memory.New(), failingPrefsRepo{err: boom}, nil, nil, nil, nil, nil, m, RetryConfig{})

		_, err := svc.UpdatePreferences(t.Context(), UpdatePreferencesRequest{UserID: "u-1"})
		require.ErrorIs(t, err, boom)
		assert.Equal(t, []string{metrics.StatusError}, m.prefsUpdate)
	})
}
