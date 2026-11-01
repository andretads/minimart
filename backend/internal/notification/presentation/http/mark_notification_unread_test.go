package http

import (
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMarkNotificationUnread(t *testing.T) {
	t.Run("marks the notification as unread", func(t *testing.T) {
		mux, repo := newTestMux(t)
		n := seedNotification(t, repo, "n-1", "o-1", "u-1")
		now := time.Now().UTC()
		n.ReadAt = &now
		require.NoError(t, repo.Save(t.Context(), n))

		rec := doAs(t, mux, http.MethodPost, "/v1/notifications/n-1/unread", "", "u-1")
		require.Equal(t, http.StatusNoContent, rec.Code)

		got, err := repo.GetByIDForUser(t.Context(), "n-1", "u-1")
		require.NoError(t, err)
		assert.Nil(t, got.ReadAt)
	})

	t.Run("missing notification returns 404", func(t *testing.T) {
		mux, _ := newTestMux(t)

		rec := doAs(t, mux, http.MethodPost, "/v1/notifications/missing/unread", "", "u-1")
		require.Equal(t, http.StatusNotFound, rec.Code)

		var env envelope
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &env))
		assert.False(t, env.Success)
		require.NotNil(t, env.Error)
		assert.Equal(t, http.StatusNotFound, env.Error.Code)
	})

	t.Run("unauthenticated returns 401", func(t *testing.T) {
		mux, _ := newTestMux(t)

		rec := doAs(t, mux, http.MethodPost, "/v1/notifications/n-1/unread", "", "")
		assert.Equal(t, http.StatusUnauthorized, rec.Code)
	})
}
