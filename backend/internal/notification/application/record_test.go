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

func TestRecordPreferenceEnforcement(t *testing.T) {
	t.Run("disabled order updates suppresses the notification", func(t *testing.T) {
		repo := memory.New()
		prefs, err := domain.NewPreferences("u-1", false, true, true, false)
		require.NoError(t, err)
		require.NoError(t, repo.Upsert(t.Context(), prefs))

		m := &countingMetrics{}
		sender := newFakeEmailSender()
		resolver := newFakeRecipientResolver()
		resolver.SetEmail("u-1", "user@example.com")
		svc := NewService(repo, repo, repo, sender, nil, nil, resolver, nil, m, RetryConfig{})

		require.NoError(t, svc.SendOrderConfirmation(t.Context(), "o-1", "u-1", 1250))
		assert.Empty(t, m.sent)
		assert.Equal(t, []string{string(domain.TypeOrderConfirmation)}, m.suppressed)

		list, err := repo.ListByUserID(t.Context(), "u-1", 1, 20)
		require.NoError(t, err)
		assert.Empty(t, list, "a suppressed notification must not be recorded")
	})

	t.Run("enabled order updates sends the notification", func(t *testing.T) {
		repo := memory.New()
		prefs, err := domain.NewPreferences("u-1", true, false, false, false)
		require.NoError(t, err)
		require.NoError(t, repo.Upsert(t.Context(), prefs))

		m := &countingMetrics{}
		sender := newFakeEmailSender()
		resolver := newFakeRecipientResolver()
		resolver.SetEmail("u-1", "user@example.com")
		svc := NewService(repo, repo, repo, sender, nil, nil, resolver, nil, m, RetryConfig{})

		require.NoError(t, svc.SendOrderConfirmation(t.Context(), "o-1", "u-1", 1250))
		assert.Equal(t, []string{metrics.StatusSuccess}, m.sent)
		assert.Empty(t, m.suppressed)
	})

	t.Run("unavailable preferences fail open", func(t *testing.T) {
		repo := memory.New()
		m := &countingMetrics{}
		sender := newFakeEmailSender()
		resolver := newFakeRecipientResolver()
		resolver.SetEmail("u-1", "user@example.com")
		svc := NewService(repo, repo, failingPrefsRepo{err: errors.New("mongo down")}, sender, nil, nil, resolver, nil, m, RetryConfig{})

		require.NoError(t, svc.SendOrderConfirmation(t.Context(), "o-1", "u-1", 1250))
		assert.Equal(t, []string{metrics.StatusSuccess}, m.sent)
		assert.Empty(t, m.suppressed)

		list, err := repo.ListByUserID(t.Context(), "u-1", 1, 20)
		require.NoError(t, err)
		assert.Len(t, list, 1)
	})
}
