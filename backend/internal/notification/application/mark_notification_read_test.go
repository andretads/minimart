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

func TestMarkNotificationRead(t *testing.T) {
	repo := memory.New()
	require.NoError(t, repo.Save(t.Context(), mustNotification(t, "n-1", "o-1", "u-1")))
	svc := NewService(repo, repo, repo, nil, nil, nil, nil, nil, metrics.Noop{}, RetryConfig{})

	t.Run("marks the notification as read", func(t *testing.T) {
		err := svc.MarkNotificationRead(t.Context(), "n-1", "u-1")
		require.NoError(t, err)

		got, err := repo.GetByIDForUser(t.Context(), "n-1", "u-1")
		require.NoError(t, err)
		require.NotNil(t, got.ReadAt)
		assert.False(t, got.ReadAt.IsZero())
	})

	t.Run("unknown id reports not found", func(t *testing.T) {
		err := svc.MarkNotificationRead(t.Context(), "missing", "u-1")
		assert.ErrorIs(t, err, domain.ErrNotificationNotFound)
	})

	t.Run("empty id is rejected", func(t *testing.T) {
		err := svc.MarkNotificationRead(t.Context(), "", "u-1")
		assert.ErrorIs(t, err, domain.ErrEmptyID)
	})

	t.Run("empty user id is rejected", func(t *testing.T) {
		err := svc.MarkNotificationRead(t.Context(), "n-1", "")
		assert.ErrorIs(t, err, domain.ErrEmptyUserID)
	})

	t.Run("other user cannot mark it read", func(t *testing.T) {
		err := svc.MarkNotificationRead(t.Context(), "n-1", "u-2")
		assert.ErrorIs(t, err, domain.ErrNotificationNotFound)
	})
}

func TestMarkNotificationUnread(t *testing.T) {
	repo := memory.New()
	n := mustNotification(t, "n-1", "o-1", "u-1")
	now := time.Now().UTC()
	n.ReadAt = &now
	require.NoError(t, repo.Save(t.Context(), n))
	svc := NewService(repo, repo, repo, nil, nil, nil, nil, nil, metrics.Noop{}, RetryConfig{})

	t.Run("clears the read timestamp", func(t *testing.T) {
		err := svc.MarkNotificationUnread(t.Context(), "n-1", "u-1")
		require.NoError(t, err)

		got, err := repo.GetByIDForUser(t.Context(), "n-1", "u-1")
		require.NoError(t, err)
		assert.Nil(t, got.ReadAt)
	})

	t.Run("unknown id reports not found", func(t *testing.T) {
		err := svc.MarkNotificationUnread(t.Context(), "missing", "u-1")
		assert.ErrorIs(t, err, domain.ErrNotificationNotFound)
	})

	t.Run("empty id is rejected", func(t *testing.T) {
		err := svc.MarkNotificationUnread(t.Context(), "", "u-1")
		assert.ErrorIs(t, err, domain.ErrEmptyID)
	})

	t.Run("empty user id is rejected", func(t *testing.T) {
		err := svc.MarkNotificationUnread(t.Context(), "n-1", "")
		assert.ErrorIs(t, err, domain.ErrEmptyUserID)
	})
}

func TestMarkAllNotificationsRead(t *testing.T) {
	repo := memory.New()
	require.NoError(t, repo.Save(t.Context(), mustNotification(t, "n-1", "o-1", "u-1")))
	require.NoError(t, repo.Save(t.Context(), mustNotification(t, "n-2", "o-2", "u-1")))
	require.NoError(t, repo.Save(t.Context(), mustNotification(t, "n-3", "o-3", "u-2")))
	svc := NewService(repo, repo, repo, nil, nil, nil, nil, nil, metrics.Noop{}, RetryConfig{})

	t.Run("marks all of the user's notifications as read", func(t *testing.T) {
		err := svc.MarkAllNotificationsRead(t.Context(), "u-1")
		require.NoError(t, err)

		for _, id := range []string{"n-1", "n-2"} {
			got, getErr := repo.GetByIDForUser(t.Context(), id, "u-1")
			require.NoError(t, getErr)
			require.NotNil(t, got.ReadAt)
		}
		got, err := repo.GetByIDForUser(t.Context(), "n-3", "u-2")
		require.NoError(t, err)
		assert.Nil(t, got.ReadAt)
	})

	t.Run("empty user id is rejected", func(t *testing.T) {
		err := svc.MarkAllNotificationsRead(t.Context(), "")
		assert.ErrorIs(t, err, domain.ErrEmptyUserID)
	})
}

func TestDeleteNotification(t *testing.T) {
	repo := memory.New()
	require.NoError(t, repo.Save(t.Context(), mustNotification(t, "n-1", "o-1", "u-1")))
	svc := NewService(repo, repo, repo, nil, nil, nil, nil, nil, metrics.Noop{}, RetryConfig{})

	t.Run("removes the notification", func(t *testing.T) {
		err := svc.DeleteNotification(t.Context(), "n-1", "u-1")
		require.NoError(t, err)

		_, err = repo.GetByIDForUser(t.Context(), "n-1", "u-1")
		assert.ErrorIs(t, err, domain.ErrNotificationNotFound)
	})

	t.Run("unknown id reports not found", func(t *testing.T) {
		err := svc.DeleteNotification(t.Context(), "missing", "u-1")
		assert.ErrorIs(t, err, domain.ErrNotificationNotFound)
	})

	t.Run("empty id is rejected", func(t *testing.T) {
		err := svc.DeleteNotification(t.Context(), "", "u-1")
		assert.ErrorIs(t, err, domain.ErrEmptyID)
	})

	t.Run("empty user id is rejected", func(t *testing.T) {
		err := svc.DeleteNotification(t.Context(), "n-1", "")
		assert.ErrorIs(t, err, domain.ErrEmptyUserID)
	})

	t.Run("other user cannot delete it", func(t *testing.T) {
		require.NoError(t, repo.Save(t.Context(), mustNotification(t, "n-2", "o-2", "u-1")))
		err := svc.DeleteNotification(t.Context(), "n-2", "u-2")
		require.ErrorIs(t, err, domain.ErrNotificationNotFound)
		_, err = repo.GetByIDForUser(t.Context(), "n-2", "u-1")
		require.NoError(t, err)
	})
}
