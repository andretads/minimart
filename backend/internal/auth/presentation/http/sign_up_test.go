package http

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSignUp(t *testing.T) {
	mux := newTestMux(t)

	t.Run("valid signup returns 201 with user info", func(t *testing.T) {
		rec := do(t, mux, http.MethodPost, "/v1/auth/signup", `{"username":"testuser","password":"password123","email":"testuser@example.com"}`)
		require.Equal(t, http.StatusCreated, rec.Code)

		var env struct {
			Success bool `json:"success"`
			Data    struct {
				ID        string `json:"id"`
				Username  string `json:"username"`
				Email     string `json:"email"`
				CreatedAt string `json:"created_at"`
			} `json:"data"`
		}
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &env))
		assert.True(t, env.Success)
		assert.NotEmpty(t, env.Data.ID)
		assert.Equal(t, "testuser", env.Data.Username)
		assert.Equal(t, "testuser@example.com", env.Data.Email)
		assert.NotEmpty(t, env.Data.CreatedAt)
	})

	t.Run("duplicate username returns 409", func(t *testing.T) {
		// First signup
		rec1 := do(t, mux, http.MethodPost, "/v1/auth/signup", `{"username":"dupuser","password":"password123","email":"dupuser@example.com"}`)
		require.Equal(t, http.StatusCreated, rec1.Code)
		// Second signup with same username
		rec2 := do(t, mux, http.MethodPost, "/v1/auth/signup", `{"username":"dupuser","password":"different123","email":"dupuser2@example.com"}`)
		require.Equal(t, http.StatusConflict, rec2.Code)

		var env struct {
			Success bool `json:"success"`
			Error   *struct {
				Code    int    `json:"code"`
				Message string `json:"message"`
			} `json:"error"`
		}
		require.NoError(t, json.Unmarshal(rec2.Body.Bytes(), &env))
		assert.False(t, env.Success)
		assert.NotNil(t, env.Error)
		assert.Equal(t, http.StatusConflict, env.Error.Code)
	})

	t.Run("duplicate email returns 409", func(t *testing.T) {
		// First signup
		rec1 := do(t, mux, http.MethodPost, "/v1/auth/signup", `{"username":"user1","password":"password123","email":"user1@example.com"}`)
		require.Equal(t, http.StatusCreated, rec1.Code)
		// Second signup with same email
		rec2 := do(t, mux, http.MethodPost, "/v1/auth/signup", `{"username":"user2","password":"different123","email":"user1@example.com"}`)
		require.Equal(t, http.StatusConflict, rec2.Code)

		var env struct {
			Success bool `json:"success"`
			Error   *struct {
				Code    int    `json:"code"`
				Message string `json:"message"`
			} `json:"error"`
		}
		require.NoError(t, json.Unmarshal(rec2.Body.Bytes(), &env))
		assert.False(t, env.Success)
		assert.NotNil(t, env.Error)
		assert.Equal(t, http.StatusConflict, env.Error.Code)
	})

	t.Run("invalid username returns 400", func(t *testing.T) {
		rec := do(t, mux, http.MethodPost, "/v1/auth/signup", `{"username":"","password":"password123","email":"test@example.com"}`)
		require.Equal(t, http.StatusBadRequest, rec.Code)
	})

	t.Run("short password returns 400", func(t *testing.T) {
		rec := do(t, mux, http.MethodPost, "/v1/auth/signup", `{"username":"testuser","password":"12345","email":"test@example.com"}`)
		require.Equal(t, http.StatusBadRequest, rec.Code)
	})

	t.Run("long username returns 400", func(t *testing.T) {
		longUsername := "a" + string(make([]byte, 200)) // 201 chars
		rec := do(t, mux, http.MethodPost, "/v1/auth/signup", `{"username":"`+longUsername+`","password":"password123","email":"test@example.com"}`)
		require.Equal(t, http.StatusBadRequest, rec.Code)
	})

	t.Run("long password returns 400", func(t *testing.T) {
		longPassword := "a" + string(make([]byte, 72)) // 73 chars
		rec := do(t, mux, http.MethodPost, "/v1/auth/signup", `{"username":"testuser","password":"`+longPassword+`","email":"test@example.com"}`)
		require.Equal(t, http.StatusBadRequest, rec.Code)
	})

	t.Run("missing username returns 400", func(t *testing.T) {
		rec := do(t, mux, http.MethodPost, "/v1/auth/signup", `{"password":"password123","email":"test@example.com"}`)
		require.Equal(t, http.StatusBadRequest, rec.Code)
	})

	t.Run("missing password returns 400", func(t *testing.T) {
		rec := do(t, mux, http.MethodPost, "/v1/auth/signup", `{"username":"testuser","email":"test@example.com"}`)
		require.Equal(t, http.StatusBadRequest, rec.Code)
	})

	t.Run("missing email returns 400", func(t *testing.T) {
		rec := do(t, mux, http.MethodPost, "/v1/auth/signup", `{"username":"testuser","password":"password123"}`)
		require.Equal(t, http.StatusBadRequest, rec.Code)
	})

	t.Run("invalid email returns 400", func(t *testing.T) {
		rec := do(t, mux, http.MethodPost, "/v1/auth/signup", `{"username":"testuser","password":"password123","email":"invalid-email"}`)
		require.Equal(t, http.StatusBadRequest, rec.Code)
	})

	t.Run("invalid JSON returns 400", func(t *testing.T) {
		rec := do(t, mux, http.MethodPost, "/v1/auth/signup", `{invalid json}`)
		require.Equal(t, http.StatusBadRequest, rec.Code)
	})
}
