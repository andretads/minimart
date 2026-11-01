package application

import (
	"minimart/internal/notification/domain"
	"minimart/internal/notification/infrastructure/memory"
	"minimart/pkg/metrics"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMarkAllNotificationsRead(t *testing.T) {
	t.Run("Empty user_id is rejected", func(t *testing.T) {
		repo := memory.New()
		svc := NewService(repo, repo, repo, nil, nil, nil, nil, nil, metrics.Noop{}, RetryConfig{})

		err := svc.MarkAllNotificationsRead(t.Context(), "")
		require.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrEmptyUserID)
	})

	t.Run("A user with no notifications gets a no-op", func(t *testing.T) {
		repo := memory.New()
		svc := NewService(repo, repo, repo, nil, nil, nil, nil, nil, metrics.Noop{}, RetryConfig{})

		err := svc.MarkAllNotificationsRead(t.Context(), "u1")
		require.NoError(t, err)
	})

	t.Run("All of the user's notifications are marked as read", func(t *testing.T) {
		repo := memory.New()
		require.NoError(t, repo.Save(t.Context(), mustNotification(t, "n1", "o1", "u1")))
		require.NoError(t, repo.Save(t.Context(), mustNotification(t, "n2", "o2", "u1")))
		svc := NewService(repo, repo, repo, nil, nil, nil, nil, nil, metrics.Noop{}, RetryConfig{})

		err := svc.MarkAllNotificationsRead(t.Context(), "u1")
		require.NoError(t, err)

		for _, id := range []string{"n1", "n2"} {
			got, getErr := repo.GetByIDForUser(t.Context(), id, "u1")
			require.NoError(t, getErr)
			require.NotNil(t, got.ReadAt)
		}
	})

	t.Run("Another user's notifications are left untouched", func(t *testing.T) {
		repo := memory.New()
		require.NoError(t, repo.Save(t.Context(), mustNotification(t, "n1", "o1", "u1")))
		require.NoError(t, repo.Save(t.Context(), mustNotification(t, "n2", "o2", "u2")))
		svc := NewService(repo, repo, repo, nil, nil, nil, nil, nil, metrics.Noop{}, RetryConfig{})

		err := svc.MarkAllNotificationsRead(t.Context(), "u1")
		require.NoError(t, err)

		got, err := repo.GetByIDForUser(t.Context(), "n2", "u2")
		require.NoError(t, err)
		assert.Nil(t, got.ReadAt)
	})
}
