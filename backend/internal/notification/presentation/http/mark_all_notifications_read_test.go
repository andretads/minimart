package http

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMarkAllNotificationsRead(t *testing.T) {
	t.Run("marks all of the user's notifications as read", func(t *testing.T) {
		mux, repo := newTestMux(t)
		seedNotification(t, repo, "n-1", "o-1", "u-1")
		seedNotification(t, repo, "n-2", "o-2", "u-1")
		seedNotification(t, repo, "n-3", "o-3", "u-2")

		rec := doAs(t, mux, http.MethodPost, "/v1/notifications/read-all", "", "u-1")
		require.Equal(t, http.StatusNoContent, rec.Code)

		for _, id := range []string{"n-1", "n-2"} {
			got, err := repo.GetByIDForUser(t.Context(), id, "u-1")
			require.NoError(t, err)
			require.NotNil(t, got.ReadAt)
		}
		got, err := repo.GetByIDForUser(t.Context(), "n-3", "u-2")
		require.NoError(t, err)
		assert.Nil(t, got.ReadAt)
	})

	t.Run("unauthenticated returns 401", func(t *testing.T) {
		mux, _ := newTestMux(t)

		rec := doAs(t, mux, http.MethodPost, "/v1/notifications/read-all", "", "")
		assert.Equal(t, http.StatusUnauthorized, rec.Code)
	})
}
