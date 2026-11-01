package message

import (
	"minimart/internal/notification/application"
	"minimart/internal/notification/domain"
	"minimart/internal/notification/infrastructure/memory"
	"minimart/pkg/kafka"
	"minimart/pkg/metrics"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDeliveryHandler(t *testing.T) {
	repo := memory.New()
	svc := application.NewService(repo, repo, repo, nil, nil, nil, nil, nil, metrics.Noop{}, application.RetryConfig{})
	handler := deliveryHandler(svc)

	t.Run("empty payload is permanent", func(t *testing.T) {
		err := handler(t.Context(), []byte(""))
		require.Error(t, err)
		assert.True(t, kafka.IsPermanent(err))
	})

	t.Run("unknown delivery is permanent", func(t *testing.T) {
		err := handler(t.Context(), []byte(`"missing-delivery"`))
		require.Error(t, err)
		assert.True(t, kafka.IsPermanent(err))
		assert.ErrorIs(t, err, domain.ErrDeliveryNotFound)
	})

	t.Run("delivery without its notification is permanent", func(t *testing.T) {
		d, err := domain.NewDelivery("orphan-delivery", "missing-notification", "u-1", domain.ChannelEmail, "resend")
		require.NoError(t, err)
		require.NoError(t, repo.SaveDelivery(t.Context(), d))

		err = handler(t.Context(), []byte(`"orphan-delivery"`))
		require.Error(t, err)
		assert.True(t, kafka.IsPermanent(err))
		assert.ErrorIs(t, err, domain.ErrNotificationNotFound)
	})
}
