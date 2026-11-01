package application

import (
	"minimart/internal/notification/domain"
	"minimart/internal/notification/infrastructure/memory"
	"minimart/pkg/metrics"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestListNotifications(t *testing.T) {
	t.Run("Missing user_id is rejected", func(t *testing.T) {
		repo := memory.New()
		svc := NewService(repo, repo, repo, nil, nil, nil, nil, nil, metrics.Noop{}, RetryConfig{})

		_, _, err := svc.ListNotifications(t.Context(), "", 1, 20)
		require.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrEmptyUserID)
	})

	t.Run("User with no notifications gets an empty page and a zero total", func(t *testing.T) {
		repo := memory.New()
		svc := NewService(repo, repo, repo, nil, nil, nil, nil, nil, metrics.Noop{}, RetryConfig{})

		got, total, err := svc.ListNotifications(t.Context(), "u1", 1, 20)
		require.NoError(t, err)
		assert.Empty(t, got)
		assert.Equal(t, 0, total)
	})

	t.Run("Only the user's notifications are returned, newest first", func(t *testing.T) {
		repo := memory.New()
		require.NoError(t, repo.Save(t.Context(), mustNotification(t, "n1", "o1", "u1")))
		require.NoError(t, repo.Save(t.Context(), mustNotification(t, "n2", "o2", "u1")))
		require.NoError(t, repo.Save(t.Context(), mustNotification(t, "n3", "o3", "u2")))
		svc := NewService(repo, repo, repo, nil, nil, nil, nil, nil, metrics.Noop{}, RetryConfig{})

		got, total, err := svc.ListNotifications(t.Context(), "u1", 1, 20)
		require.NoError(t, err)
		assert.Len(t, got, 2)
		assert.Equal(t, 2, total)
		assert.Equal(t, "n2", got[0].ID)
	})

	t.Run("Pagination returns the requested page", func(t *testing.T) {
		repo := memory.New()
		require.NoError(t, repo.Save(t.Context(), mustNotification(t, "n1", "o1", "u1")))
		require.NoError(t, repo.Save(t.Context(), mustNotification(t, "n2", "o2", "u1")))
		require.NoError(t, repo.Save(t.Context(), mustNotification(t, "n3", "o3", "u1")))
		svc := NewService(repo, repo, repo, nil, nil, nil, nil, nil, metrics.Noop{}, RetryConfig{})

		got, _, err := svc.ListNotifications(t.Context(), "u1", 2, 2)
		require.NoError(t, err)
		assert.Len(t, got, 1)
		assert.Equal(t, "n1", got[0].ID)
	})
}
