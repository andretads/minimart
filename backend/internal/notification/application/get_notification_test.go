package application

import (
	"minimart/internal/notification/domain"
	"minimart/internal/notification/infrastructure/memory"
	"minimart/pkg/metrics"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetNotification(t *testing.T) {
	repo := memory.New()
	require.NoError(t, repo.Save(t.Context(), mustNotification(t, "n1", "o1", "u1")))
	svc := NewService(repo, repo, repo, nil, nil, nil, nil, nil, metrics.Noop{}, RetryConfig{})

	t.Run("Empty notification id is rejected", func(t *testing.T) {
		_, err := svc.GetNotification(t.Context(), "", "u1")
		require.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrEmptyID)
	})

	t.Run("Empty user_id is rejected", func(t *testing.T) {
		_, err := svc.GetNotification(t.Context(), "n1", "")
		require.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrEmptyUserID)
	})

	t.Run("Notification does not exist", func(t *testing.T) {
		_, err := svc.GetNotification(t.Context(), "missing", "u1")
		require.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrNotificationNotFound)
	})

	t.Run("Another user's notification is not found", func(t *testing.T) {
		_, err := svc.GetNotification(t.Context(), "n1", "u2")
		require.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrNotificationNotFound)
	})

	t.Run("Notification got returned with no problems", func(t *testing.T) {
		got, err := svc.GetNotification(t.Context(), "n1", "u1")
		require.NoError(t, err)
		assert.Equal(t, "n1", got.ID)
		assert.Equal(t, domain.TypeOrderConfirmation, got.Type)
		assert.Equal(t, "o1", got.OrderID)
		assert.Equal(t, "u1", got.UserID)
		assert.False(t, got.SentAt.IsZero())
	})
}
