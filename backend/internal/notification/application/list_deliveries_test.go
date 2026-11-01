package application

import (
	"minimart/internal/notification/domain"
	"minimart/internal/notification/infrastructure/memory"
	"minimart/pkg/metrics"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestListDeliveries(t *testing.T) {
	repo := memory.New()
	prefs, _ := domain.NewPreferences("u-1", true, false, false, false)
	require.NoError(t, repo.Upsert(t.Context(), prefs))
	resolver := newFakeRecipientResolver()
	resolver.SetEmail("u-1", "user@example.com")
	svc := NewService(repo, repo, repo, newFakeEmailSender(), nil, nil, resolver, nil, metrics.Noop{}, RetryConfig{})

	n, err := domain.NewOrderConfirmation(newID(), "o-1", "u-1", 1250)
	require.NoError(t, err)
	_, deliveries, err := svc.IngestNotification(t.Context(), n, nil)
	require.NoError(t, err)
	require.Len(t, deliveries, 1)

	got, err := svc.ListDeliveries(t.Context(), n.ID, "u-1")
	require.NoError(t, err)
	require.Len(t, got, 1)
	assert.Equal(t, deliveries[0].ID, got[0].ID)

	t.Run("empty notification id is rejected", func(t *testing.T) {
		_, err := svc.ListDeliveries(t.Context(), "", "u-1")
		assert.ErrorIs(t, err, domain.ErrEmptyNotificationID)
	})

	t.Run("other user cannot list deliveries", func(t *testing.T) {
		_, err := svc.ListDeliveries(t.Context(), n.ID, "u-2")
		assert.ErrorIs(t, err, domain.ErrNotificationNotFound)
	})
}
