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

func TestGetPreferences(t *testing.T) {
	t.Run("Empty user_id is rejected", func(t *testing.T) {
		repo := memory.New()
		svc := NewService(repo, repo, repo, nil, nil, nil, nil, nil, metrics.Noop{}, RetryConfig{})

		_, err := svc.GetPreferences(t.Context(), "")
		require.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrEmptyUserID)
	})

	t.Run("Preferences storage failure is propagated", func(t *testing.T) {
		boom := errors.New("mongo down")
		svc := NewService(memory.New(), memory.New(), failingPrefsRepo{err: boom}, nil, nil, nil, nil, nil, metrics.Noop{}, RetryConfig{})

		_, err := svc.GetPreferences(t.Context(), "u1")
		require.Error(t, err)
		assert.ErrorIs(t, err, boom)
	})

	t.Run("A user with no saved preferences gets the defaults", func(t *testing.T) {
		repo := memory.New()
		svc := NewService(repo, repo, repo, nil, nil, nil, nil, nil, metrics.Noop{}, RetryConfig{})

		got, err := svc.GetPreferences(t.Context(), "u1")
		require.NoError(t, err)
		assert.True(t, got.EmailEnabled)
		assert.False(t, got.SMSEnabled)
		assert.False(t, got.PushEnabled)
		assert.False(t, got.MarketingEnabled)
		assert.True(t, got.UpdatedAt.IsZero())
	})

	t.Run("Saved preferences are returned unchanged", func(t *testing.T) {
		repo := memory.New()
		stored, err := domain.NewPreferences("u1", false, true, false, true)
		require.NoError(t, err)
		require.NoError(t, repo.Upsert(t.Context(), stored))
		svc := NewService(repo, repo, repo, nil, nil, nil, nil, nil, metrics.Noop{}, RetryConfig{})

		got, err := svc.GetPreferences(t.Context(), "u1")
		require.NoError(t, err)
		assert.False(t, got.EmailEnabled)
		assert.True(t, got.SMSEnabled)
		assert.False(t, got.PushEnabled)
		assert.True(t, got.MarketingEnabled)
		assert.False(t, got.UpdatedAt.IsZero())
	})
}
