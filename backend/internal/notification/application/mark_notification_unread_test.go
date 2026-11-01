package application

import (
	"minimart/internal/notification/domain"
	"minimart/internal/notification/infrastructure/memory"
	"minimart/pkg/metrics"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMarkNotificationUnread(t *testing.T) {
	t.Run("Empty notification id is rejected", func(t *testing.T) {
		repo := memory.New()
		svc := NewService(repo, repo, repo, nil, nil, nil, nil, nil, metrics.Noop{}, RetryConfig{})

		err := svc.MarkNotificationUnread(t.Context(), "", "u1")
		require.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrEmptyID)
	})

	t.Run("Empty user_id is rejected", func(t *testing.T) {
		repo := memory.New()
		svc := NewService(repo, repo, repo, nil, nil, nil, nil, nil, metrics.Noop{}, RetryConfig{})

		err := svc.MarkNotificationUnread(t.Context(), "n1", "")
		require.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrEmptyUserID)
	})

	t.Run("Notification does not exist", func(t *testing.T) {
		repo := memory.New()
		svc := NewService(repo, repo, repo, nil, nil, nil, nil, nil, metrics.Noop{}, RetryConfig{})

		err := svc.MarkNotificationUnread(t.Context(), "missing", "u1")
		require.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrNotificationNotFound)
	})

	t.Run("Another user's notification is not found", func(t *testing.T) {
		repo := memory.New()
		n := mustNotification(t, "n1", "o1", "u1")
		now := time.Now().UTC()
		n.ReadAt = &now
		require.NoError(t, repo.Save(t.Context(), n))
		svc := NewService(repo, repo, repo, nil, nil, nil, nil, nil, metrics.Noop{}, RetryConfig{})

		err := svc.MarkNotificationUnread(t.Context(), "n1", "u2")
		require.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrNotificationNotFound)
	})

	t.Run("The notification is marked as unread", func(t *testing.T) {
		repo := memory.New()
		n := mustNotification(t, "n1", "o1", "u1")
		now := time.Now().UTC()
		n.ReadAt = &now
		require.NoError(t, repo.Save(t.Context(), n))
		svc := NewService(repo, repo, repo, nil, nil, nil, nil, nil, metrics.Noop{}, RetryConfig{})

		err := svc.MarkNotificationUnread(t.Context(), "n1", "u1")
		require.NoError(t, err)

		got, err := repo.GetByIDForUser(t.Context(), "n1", "u1")
		require.NoError(t, err)
		assert.Nil(t, got.ReadAt)
	})
}
