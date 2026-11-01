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
	require.NoError(t, repo.Save(t.Context(), mustNotification(t, "n-1", "o-1", "u-1")))
	svc := NewService(repo, repo, repo, nil, nil, nil, nil, nil, metrics.Noop{}, RetryConfig{})

	t.Run("returns the stored notification", func(t *testing.T) {
		got, err := svc.GetNotification(t.Context(), "n-1", "u-1")
		require.NoError(t, err)
		assert.Equal(t, "n-1", got.ID)
		assert.Equal(t, "o-1", got.OrderID)
	})

	t.Run("unknown id reports not found", func(t *testing.T) {
		_, err := svc.GetNotification(t.Context(), "missing", "u-1")
		assert.ErrorIs(t, err, domain.ErrNotificationNotFound)
	})
}

func mustNotification(t *testing.T, id, orderID, userID string) domain.Notification {
	t.Helper()
	n, err := domain.NewOrderConfirmation(id, orderID, userID, 1000)
	require.NoError(t, err)
	return n
}
