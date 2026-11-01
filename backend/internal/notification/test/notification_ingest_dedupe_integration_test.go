//go:build integration

package test

import (
	"minimart/internal/notification/application"
	"minimart/internal/notification/domain"
	"minimart/pkg/metrics"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIngestNotificationDuplicateEventKeepsOneNotification(t *testing.T) {
	ctx := t.Context()
	repo := newTestRepo(t)
	deliveries := newTestDeliveryRepo(t)
	prefs := newTestPrefsRepo(t)

	resolver := newFakeRecipientResolver()
	resolver.SetEmail("it-user-dedupe", "dedupe@example.com")
	svc := application.NewService(repo, deliveries, prefs, newFakeEmailSender(), nil, nil, resolver, nil, metrics.Noop{}, application.RetryConfig{})

	first := mustNotification(t, "it-dedupe-1", "it-order-dedupe", "it-user-dedupe")
	persisted, created, err := svc.IngestNotification(ctx, first, nil)
	require.NoError(t, err)
	require.Len(t, created, 1)

	duplicate := mustNotification(t, "it-dedupe-2", "it-order-dedupe", "it-user-dedupe")
	got, reused, err := svc.IngestNotification(ctx, duplicate, nil)
	require.NoError(t, err)

	assert.Equal(t, persisted.ID, got.ID, "the duplicate event must reuse the stored notification")
	assert.Len(t, reused, 1)

	all, err := repo.ListByUserID(ctx, "it-user-dedupe", 1, 20)
	require.NoError(t, err)
	assert.Len(t, all, 1)

	orphans, err := deliveries.ListByNotification(ctx, duplicate.ID)
	require.NoError(t, err)
	assert.Empty(t, orphans, "a duplicate event must not create deliveries for an unsaved notification")

	stored, err := deliveries.ListByNotification(ctx, persisted.ID)
	require.NoError(t, err)
	assert.Len(t, stored, 1)

	_, err = repo.GetByID(ctx, duplicate.ID)
	assert.ErrorIs(t, err, domain.ErrNotificationNotFound)
}
