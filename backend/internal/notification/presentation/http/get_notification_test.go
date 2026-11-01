package http

import (
	"encoding/json"
	"minimart/internal/notification/domain"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetNotification(t *testing.T) {
	mux, repo := newTestMux(t)
	want := seedNotification(t, repo, "n-1", "o-1", "u-1")

	t.Run("found returns the notification", func(t *testing.T) {
		rec := do(t, mux, "/v1/notifications/n-1")
		require.Equal(t, http.StatusOK, rec.Code)

		var env struct {
			Data notificationResponse `json:"data"`
		}
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &env))
		assert.Equal(t, want.ID, env.Data.ID)
		assert.Equal(t, "order_confirmation", env.Data.Type)
		assert.Equal(t, "u-1", env.Data.UserID)
		assert.Equal(t, "o-1", env.Data.OrderID)
		assert.Equal(t, want.Subject, env.Data.Subject)
		assert.Equal(t, want.Body, env.Data.Body)
		assert.False(t, env.Data.SentAt.IsZero())
		assert.False(t, env.Data.CreatedAt.IsZero())
	})

	t.Run("missing returns 404", func(t *testing.T) {
		rec := do(t, mux, "/v1/notifications/missing")
		require.Equal(t, http.StatusNotFound, rec.Code)

		var env envelope
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &env))
		assert.False(t, env.Success)
		require.NotNil(t, env.Error)
		assert.Equal(t, http.StatusNotFound, env.Error.Code)
		assert.Equal(t, domain.ErrNotificationNotFound.Error(), env.Error.Message)
	})
}
