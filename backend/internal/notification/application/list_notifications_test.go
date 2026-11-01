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
	repo := memory.New()
	require.NoError(t, repo.Save(t.Context(), mustNotification(t, "n-1", "o-1", "u-1")))
	require.NoError(t, repo.Save(t.Context(), mustNotification(t, "n-2", "o-2", "u-1")))
	require.NoError(t, repo.Save(t.Context(), mustNotification(t, "n-3", "o-3", "u-2")))
	svc := NewService(repo, repo, repo, nil, nil, nil, nil, nil, metrics.Noop{}, RetryConfig{})

	t.Run("returns only the user's notifications", func(t *testing.T) {
		got, _, err := svc.ListNotifications(t.Context(), "u-1", 1, 20)
		require.NoError(t, err)
		require.Len(t, got, 2)
	})

	t.Run("empty user id is rejected", func(t *testing.T) {
		_, _, err := svc.ListNotifications(t.Context(), "", 1, 20)
		assert.ErrorIs(t, err, domain.ErrEmptyUserID)
	})
}
