package http

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDeleteNotification(t *testing.T) {
	t.Run("deletes the notification", func(t *testing.T) {
		mux, repo := newTestMux(t)
		seedNotification(t, repo, "n-1", "o-1", "u-1")

		rec := doAs(t, mux, http.MethodDelete, "/v1/notifications/n-1", "", "u-1")
		require.Equal(t, http.StatusNoContent, rec.Code)

		_, err := repo.GetByIDForUser(t.Context(), "n-1", "u-1")
		require.Error(t, err)
	})

	t.Run("missing notification returns 404", func(t *testing.T) {
		mux, _ := newTestMux(t)

		rec := doAs(t, mux, http.MethodDelete, "/v1/notifications/missing", "", "u-1")
		require.Equal(t, http.StatusNotFound, rec.Code)

		var env envelope
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &env))
		assert.False(t, env.Success)
		require.NotNil(t, env.Error)
		assert.Equal(t, http.StatusNotFound, env.Error.Code)
	})

	t.Run("unauthenticated returns 401", func(t *testing.T) {
		mux, repo := newTestMux(t)
		seedNotification(t, repo, "n-1", "o-1", "u-1")

		rec := doAs(t, mux, http.MethodDelete, "/v1/notifications/n-1", "", "")
		assert.Equal(t, http.StatusUnauthorized, rec.Code)

		_, err := repo.GetByIDForUser(t.Context(), "n-1", "u-1")
		require.NoError(t, err)
	})
}
