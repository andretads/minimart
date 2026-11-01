package application

import (
	"minimart/internal/notification/domain"
	"minimart/internal/notification/infrastructure/memory"
	"minimart/pkg/metrics"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDeleteNotification(t *testing.T) {
	t.Run("Empty notification id is rejected", func(t *testing.T) {
		repo := memory.New()
		svc := NewService(repo, repo, repo, nil, nil, nil, nil, nil, metrics.Noop{}, RetryConfig{})

		err := svc.DeleteNotification(t.Context(), "", "u1")
		require.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrEmptyID)
	})

	t.Run("Empty user_id is rejected", func(t *testing.T) {
		repo := memory.New()
		svc := NewService(repo, repo, repo, nil, nil, nil, nil, nil, metrics.Noop{}, RetryConfig{})

		err := svc.DeleteNotification(t.Context(), "n1", "")
		require.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrEmptyUserID)
	})

	t.Run("Notification does not exist", func(t *testing.T) {
		repo := memory.New()
		svc := NewService(repo, repo, repo, nil, nil, nil, nil, nil, metrics.Noop{}, RetryConfig{})

		err := svc.DeleteNotification(t.Context(), "missing", "u1")
		require.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrNotificationNotFound)
	})

	t.Run("Another user's notification is not found", func(t *testing.T) {
		repo := memory.New()
		require.NoError(t, repo.Save(t.Context(), mustNotification(t, "n1", "o1", "u1")))
		svc := NewService(repo, repo, repo, nil, nil, nil, nil, nil, metrics.Noop{}, RetryConfig{})

		err := svc.DeleteNotification(t.Context(), "n1", "u2")
		require.Error(t, err)
		require.ErrorIs(t, err, domain.ErrNotificationNotFound)

		_, err = repo.GetByIDForUser(t.Context(), "n1", "u1")
		require.NoError(t, err)
	})

	t.Run("The notification is deleted", func(t *testing.T) {
		repo := memory.New()
		require.NoError(t, repo.Save(t.Context(), mustNotification(t, "n1", "o1", "u1")))
		svc := NewService(repo, repo, repo, nil, nil, nil, nil, nil, metrics.Noop{}, RetryConfig{})

		err := svc.DeleteNotification(t.Context(), "n1", "u1")
		require.NoError(t, err)

		_, err = repo.GetByIDForUser(t.Context(), "n1", "u1")
		assert.ErrorIs(t, err, domain.ErrNotificationNotFound)
	})
}
